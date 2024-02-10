package testaroli

import (
	"reflect"
	"testing"
)

type testCase struct {
	name     string
	expected reflect.Value
	actual   reflect.Value
	equal    bool
	message  string
}

func TestBasicTypes(t *testing.T) {
	cases := []testCase{
		{
			"equal bool", reflect.ValueOf(true), reflect.ValueOf(true), true, "",
		},
		{
			"non-equal bool", reflect.ValueOf(true), reflect.ValueOf(false), false, "",
		},
		{
			"equal int", reflect.ValueOf(int(1)), reflect.ValueOf(int(1)), true, "",
		},
		{
			"non-equal int", reflect.ValueOf(int(1)), reflect.ValueOf(int(2)), false, "",
		},
		{
			"equal int8", reflect.ValueOf(int8(1)), reflect.ValueOf(int8(1)), true, "",
		},
		{
			"non-equal int8", reflect.ValueOf(int8(1)), reflect.ValueOf(int8(2)), false, "",
		},
		{
			"equal int16", reflect.ValueOf(int16(1)), reflect.ValueOf(int16(1)), true, "",
		},
		{
			"non-equal int16", reflect.ValueOf(int16(1)), reflect.ValueOf(int16(2)), false, "",
		},
		{
			"equal int32", reflect.ValueOf(int32(1)), reflect.ValueOf(int32(1)), true, "",
		},
		// rune is an alias for int32 so no need to test it separately
		{
			"non-equal int32", reflect.ValueOf(int32(1)), reflect.ValueOf(int32(2)), false, "",
		},
		{
			"equal int64", reflect.ValueOf(int64(1)), reflect.ValueOf(int64(1)), true, "",
		},
		{
			"non-equal int64", reflect.ValueOf(int64(1)), reflect.ValueOf(int64(2)), false, "",
		},
		{
			"equal uint", reflect.ValueOf(uint(1)), reflect.ValueOf(uint(1)), true, "",
		},
		{
			"non-equal uint", reflect.ValueOf(uint(1)), reflect.ValueOf(uint(2)), false, "",
		},
		{
			"equal uint8", reflect.ValueOf(uint8(1)), reflect.ValueOf(uint8(1)), true, "",
		},
		{
			"non-equal uint8", reflect.ValueOf(uint8(1)), reflect.ValueOf(uint8(2)), false, "",
		},
		{
			"equal uint16", reflect.ValueOf(uint16(1)), reflect.ValueOf(uint16(1)), true, "",
		},
		{
			"non-equal uint16", reflect.ValueOf(uint16(1)), reflect.ValueOf(uint16(2)), false, "",
		},
		{
			"equal uint32", reflect.ValueOf(uint32(1)), reflect.ValueOf(uint32(1)), true, "",
		},
		{
			"non-equal uint32", reflect.ValueOf(uint32(1)), reflect.ValueOf(uint32(2)), false, "",
		},
		{
			"equal uint64", reflect.ValueOf(uint64(1)), reflect.ValueOf(uint64(1)), true, "",
		},
		{
			"non-equal uint64", reflect.ValueOf(uint64(1)), reflect.ValueOf(uint64(2)), false, "",
		},
		{
			"equal float32", reflect.ValueOf(float32(1.5)), reflect.ValueOf(float32(1.5)), true, "",
		},
		{
			"non-equal float32", reflect.ValueOf(float32(1.5)), reflect.ValueOf(float32(2.5)), false, "",
		},
		{
			"equal float64", reflect.ValueOf(float64(1.5)), reflect.ValueOf(float64(1.5)), true, "",
		},
		{
			"non-equal float64", reflect.ValueOf(float64(1.5)), reflect.ValueOf(float64(2.5)), false, "",
		},
		{
			"equal complex64", reflect.ValueOf(complex(1, 2)), reflect.ValueOf(complex(1, 2)), true, "",
		},
		{
			"non-equal complex64", reflect.ValueOf(complex64(1 + 2i)), reflect.ValueOf(complex64(1 + 4i)), false, "",
		},
		{
			"equal complex128", reflect.ValueOf(complex(1, 2)), reflect.ValueOf(complex(1, 2)), true, "",
		},
		{
			"non-equal complex128", reflect.ValueOf(complex(1, 2)), reflect.ValueOf(complex(1, 4)), false, "",
		},
		{
			"equal string", reflect.ValueOf("foo"), reflect.ValueOf("foo"), true, "",
		},
		{
			"non-equal string", reflect.ValueOf("foo"), reflect.ValueOf("bar"), false, "",
		},
		{
			"different types", reflect.ValueOf(1), reflect.ValueOf("bar"), false, "",
		},
	}

	for _, c := range cases {
		res, _ := equal(c.actual, c.expected)
		if res != c.equal {
			t.Errorf("Case '%s' result mismatched", c.name)
		}
	}
}

func TestCompositeTypes(t *testing.T) {
	chan1 := make(chan int)
	chan2 := make(chan int)
	ptr1 := new(int)
	ptr2 := new(int)
	ptr3 := new(int)
	*ptr1 = 1
	*ptr2 = 1
	*ptr3 = 3
	arr1 := [...]int{1, 2}
	arr2 := [...]int{1, 2}
	arr3 := [...]int{1, 2, 3}
	arr4 := [...]int{1, 3}
	str1 := struct {
		a int
		b string
	}{5, "foo"}
	str2 := struct {
		a int
		b string
	}{5, "foo"}
	str3 := struct {
		a int
		b string
	}{5, "bar"}
	str4 := struct {
		a int
		b string
		c bool
	}{5, "foo", true}
	map1 := map[int]string{1: "foo", 2: "bar"}
	map2 := map[int]string{1: "foo", 2: "bar"}
	map3 := map[int]string{1: "foo", 3: "bar"}
	map4 := map[int]string{1: "foo", 2: "baz"}
	map5 := map[int]string{1: "foo", 2: "bar", 3: "baz"}
	map6 := map[int]int{1: 42}
	sl1 := []int{1, 2}
	sl2 := []int{1, 2}
	sl3 := []int{1, 2, 3}
	sl4 := []int{2, 1}
	sl5 := []float32{1, 2}
	cases := []testCase{
		// channel can only match to itself
		{
			"same channel", reflect.ValueOf(chan1), reflect.ValueOf(chan1), true, "",
		},
		{
			"different channel", reflect.ValueOf(chan1), reflect.ValueOf(chan2), false, "",
		},
		{
			"same pointer", reflect.ValueOf(ptr1), reflect.ValueOf(ptr1), true, "",
		},
		{
			"pointer with the same value", reflect.ValueOf(ptr1), reflect.ValueOf(ptr2), true, "",
		},
		{
			"pointer with the different value", reflect.ValueOf(ptr1), reflect.ValueOf(ptr3), false, "",
		},
		{
			"same array", reflect.ValueOf(arr1), reflect.ValueOf(arr1), true, "",
		},
		{
			"matching array", reflect.ValueOf(arr1), reflect.ValueOf(arr2), true, "",
		},
		{
			"array of diff length", reflect.ValueOf(arr1), reflect.ValueOf(arr3), false, "",
		},
		{
			"non-matching array", reflect.ValueOf(arr1), reflect.ValueOf(arr4), false, "",
		},
		{
			"same struct", reflect.ValueOf(str1), reflect.ValueOf(str1), true, "",
		},
		{
			"matching struct", reflect.ValueOf(str1), reflect.ValueOf(str2), true, "",
		},
		{
			"struct of diff type", reflect.ValueOf(str1), reflect.ValueOf(str4), false, "",
		},
		{
			"struct with different fields", reflect.ValueOf(str1), reflect.ValueOf(str3), false, "",
		},
		{
			"same map", reflect.ValueOf(map1), reflect.ValueOf(map1), true, "",
		},
		{
			"matching map", reflect.ValueOf(map1), reflect.ValueOf(map2), true, "",
		},
		{
			"map with different keys", reflect.ValueOf(map1), reflect.ValueOf(map3), false, "",
		},
		{
			"map with different values", reflect.ValueOf(map1), reflect.ValueOf(map4), false, "",
		},
		{
			"map with extra keys", reflect.ValueOf(map1), reflect.ValueOf(map5), false, "",
		},
		{
			"map of different type", reflect.ValueOf(map1), reflect.ValueOf(map6), false, "",
		},
		{
			"func", reflect.ValueOf(func() {}), reflect.ValueOf(func() {}), false, "",
		},
		{
			"same slice", reflect.ValueOf(sl1), reflect.ValueOf(sl1), true, "",
		},
		{
			"matching slice", reflect.ValueOf(sl1), reflect.ValueOf(sl2), true, "",
		},
		{
			"longer slice", reflect.ValueOf(sl1), reflect.ValueOf(sl3), false, "",
		},
		{
			"slice with different order of elements", reflect.ValueOf(sl1), reflect.ValueOf(sl4), false, "",
		},
		{
			"slice of different base type", reflect.ValueOf(sl1), reflect.ValueOf(sl5), false, "",
		},
	}

	for _, c := range cases {
		res, _ := equal(c.actual, c.expected)
		if res != c.equal {
			t.Errorf("Case '%s' result mismatched", c.name)
		}
	}
}
