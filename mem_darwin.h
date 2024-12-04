// This file is part of Testaroli project, available at https://github.com/qrdl/testaroli
// Copyright (c) 2024 Ilya Caramishev. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at https://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

#ifdef __APPLE__
#include <stdlib.h>
#include <stdint.h>
#include <unistd.h>
#include <stdio.h>
#include <mach/mach_init.h>
#include <mach/vm_map.h>
#include <mach/mach_vm.h>
#include <mach/task.h>
#include <mach/thread_act.h>

#define CHECK_ERR(MSG) if (ret != 0) { fprintf(stderr, "%d: %s: %d\n", __LINE__, MSG, ret); return ret; }

mach_vm_address_t text_segment, temp_segment;

typedef int (* mem_patch)(mach_vm_address_t, mach_vm_size_t, mach_vm_address_t);
static int recreate_text_segment(mach_vm_address_t text, mach_vm_size_t size, mach_vm_address_t tmp);
static int overwrite(mach_vm_address_t src, mach_vm_size_t size, mach_vm_address_t dest);
static int suspend_other_threads();
static int resume_other_threads();

int make_text_writable() {
    task_t task;
    kern_return_t ret = task_for_pid(mach_task_self(), getpid(), &task);
    CHECK_ERR("task_for_pid");

    // TEXT segment
    text_segment = 0x100000000; // segment expected initial address
    mach_vm_size_t text_size = 0;
    vm_region_basic_info_data_64_t info;
    mach_msg_type_number_t info_count = VM_REGION_BASIC_INFO_COUNT_64;
    memory_object_name_t object;
    ret = mach_vm_region(task,  &text_segment, &text_size, VM_REGION_BASIC_INFO_64, (vm_region_info_t)&info, &info_count, &object);
    CHECK_ERR("mach_vm_region");

    // DATA_CONST segment
    mach_vm_address_t data_segment = text_segment + text_size;  // right after TEXT segment
    mach_vm_size_t data_size = 0;
    ret = mach_vm_region(task,  &data_segment, &data_size, VM_REGION_BASIC_INFO_64, (vm_region_info_t)&info, &info_count, &object);
    CHECK_ERR("mach_vm_region");
    if (data_segment != text_segment + text_size) {
        // should never happen
        fprintf(stderr, "DATA segment doesn't follow TEXT segment, cannot continue");
        return 1;
    }

    // DATA segment
    mach_vm_address_t data2_segment = data_segment + data_size;  // right after DATA_CONST segment
    mach_vm_size_t data2_size = 0;
    ret = mach_vm_region(task,  &data2_segment, &data2_size, VM_REGION_BASIC_INFO_64, (vm_region_info_t)&info, &info_count, &object);
    CHECK_ERR("mach_vm_region");

    // allocate new VM segment of the size of TEXT, DATA_CONST and DATA segments combined
    mach_vm_size_t temp_size = text_size + data_size + data2_size;
    ret = mach_vm_allocate(task, &temp_segment, temp_size, VM_FLAGS_ANYWHERE);
    CHECK_ERR("mach_vm_allocate");
    ret = mach_vm_protect(task, temp_segment, temp_size, 0, VM_PROT_READ|VM_PROT_WRITE);
    CHECK_ERR("mach_vm_protect");

    // copy TEXT, DATA_CONST and DATA segments to TEMP segment (to keep relative references to vars in DATA_CONST and DATA segments correct)
    ret = mach_vm_copy(task, text_segment, text_size, temp_segment);
    CHECK_ERR("mach_vm_copy");
    ret = mach_vm_copy(task, data_segment, data_size, temp_segment+text_size);
    CHECK_ERR("mach_vm_copy");
    ret = mach_vm_copy(task, data2_segment, data2_size, temp_segment+text_size+data_size);
    CHECK_ERR("mach_vm_copy");

    // make TEMP segment executable
    ret = mach_vm_protect(task, temp_segment, temp_size, 0, VM_PROT_READ|VM_PROT_EXECUTE);
    CHECK_ERR("mach_vm_protect");

    // execute recreate_text_segment() in TEMP segment (to allow destructive actions on TEXT)
    mem_patch func = recreate_text_segment;
    func = func - text_segment + temp_segment;  // recreate_text_segment() address in TEMP segment
    ret = func(text_segment, text_size, temp_segment);
    CHECK_ERR("recreate_text_segment");

    // back into re-created TEXT segment

    __builtin___clear_cache((char *)text_segment, (char *)(text_segment + text_size));

    return 0;
}

// this is TEXT segment trampoline to execute overwrite() in TEMP segment
// macOS doesn't allow to change protection of TEXT segment while executing
// code in that segment, so need to jump to other segment for a while to make a change
int overwrite_prolog(uint64_t func_addr, uint64_t buf, uint64_t bufsize) {
    mem_patch func = overwrite;
    func = func - text_segment + temp_segment;  // overwrite() address in TEMP segment
    return func(buf, bufsize, func_addr);   // error to be reported by the caller
}

// this function is executed in TEMP segment
// it destroys original TEXT segment, creates new one in the same place and copies
// the original TEXT content from TEMP segment to new TEXT
// this function must not update any globals because TEMP segment is not writable!
static int recreate_text_segment(mach_vm_address_t text, mach_vm_size_t size, mach_vm_address_t tmp) {
    task_t task;
    kern_return_t ret = task_for_pid(mach_task_self(), getpid(), &task);
    CHECK_ERR("task_for_pid");

    // need to suspend all other threads to prevent them accessing TEXT while it is not there
    ret = suspend_other_threads();
    CHECK_ERR("suspend threads");

    ret = mach_vm_deallocate(task, text, size);
    CHECK_ERR("mach_vm_deallocate");

    mach_vm_address_t new_text = text;
    ret = mach_vm_allocate(task, &new_text, size, VM_FLAGS_FIXED);
    CHECK_ERR("mach_vm_allocate");
    if (new_text != text) { // should never happen
        fprintf(stderr, "New TEXT has different address, cannot continue");
        return 1;
    }

    ret = mach_vm_protect(task, text, size, 0, VM_PROT_READ|VM_PROT_WRITE);
    CHECK_ERR("mach_vm_protect");

    ret = mach_vm_copy(task, tmp, size, text);
    CHECK_ERR("mach_vm_copy");

    ret = mach_vm_protect(task, text, size, 0, VM_PROT_READ|VM_PROT_EXECUTE);
    CHECK_ERR("mach_vm_protect");

    ret = resume_other_threads();
    CHECK_ERR("resume threads");

    return 0;
}

// this function is executed in TEMP segment, but addresses must be in TEXT segment
// this function must not update any globals because TEMP segment is not writable!
static int overwrite(mach_vm_address_t src, mach_vm_size_t size, mach_vm_address_t dest) {
    task_t task;
    kern_return_t ret = task_for_pid(mach_task_self(), getpid(), &task);
    CHECK_ERR("task_for_pid");

    ret = mach_vm_protect(task, dest, size, 0, VM_PROT_READ|VM_PROT_WRITE);
    CHECK_ERR("mach_vm_protect");

    ret = mach_vm_copy(task, src, size, dest);
    CHECK_ERR("mach_vm_copy");

    ret = mach_vm_protect(task, dest, size, 0, VM_PROT_READ|VM_PROT_EXECUTE);
    CHECK_ERR("mach_vm_protect");

    return 0;
}

// this function must not update any globals because TEMP segment is not writable!
static int suspend_other_threads() {
    task_t task;
    int ret = task_for_pid(mach_task_self(), getpid(), &task);
    CHECK_ERR("task_for_pid");

    thread_act_array_t threads;
    mach_msg_type_number_t thread_count;
    ret = task_threads(task, &threads, &thread_count);
    CHECK_ERR("task_threads");

    // skip thread 0 which is the current one
    for (int i = 1; i < thread_count; i++) {
        ret = thread_suspend(threads[i]);
        if (ret != 0)
            fprintf(stderr, "suspend returned %d for thread %d\n", ret, i);
    }

    return 0;
}

// this function must not update any globals because TEMP segment is not writable!
static int resume_other_threads() {
    task_t task;
    int ret = task_for_pid(mach_task_self(), getpid(), &task);
    CHECK_ERR("task_for_pid");

    thread_act_array_t threads;
    mach_msg_type_number_t thread_count;
    ret = task_threads(task, &threads, &thread_count);
    CHECK_ERR("task_threads");

    // skip thread 0 which is the current one
    for (int i = 1; i < thread_count; i++) {
        ret = thread_resume(threads[i]);
        if (ret != 0)
            fprintf(stderr, "resume returned %d for thread %d\n", ret, i);
    }

    return 0;
}

#endif
