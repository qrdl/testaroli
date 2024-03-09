package testaroli

import (
	"fmt"
	"reflect"
)

// standard reflect.Value.Equal has several issues:
// - it compares pointers only as addresses
// - it doesn't compare maps
// - it doesn't compare slices
// - it doesn't explain what exactly has failed
// - it panics
// so I've rolled my own, based on reflect's implementation
func equal(a, e reflect.Value) (bool, string) {
	if a.Kind() == reflect.Interface {
		a = a.Elem()
	}
	if e.Kind() == reflect.Interface {
		e = e.Elem()
	}

	if !a.IsValid() || !e.IsValid() {
		return a.IsValid() == e.IsValid(), "cannot compare invalid value with valid one"
	}

	if a.Kind() != e.Kind() || a.Type() != e.Type() {
		return false, fmt.Sprintf("actual type '%s' differs from expected '%s'", a.Type(), e.Type())
	}

	switch a.Kind() {
	case reflect.Bool:
		return a.Bool() == e.Bool(), ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return a.Int() == e.Int(), ""
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return a.Uint() == e.Uint(), ""
	case reflect.Float32, reflect.Float64:
		return a.Float() == e.Float(), ""
	case reflect.Complex64, reflect.Complex128:
		return a.Complex() == e.Complex(), ""
	case reflect.String:
		return a.String() == e.String(), ""
	case reflect.Chan:
		return a.Pointer() == e.Pointer(), ""
	case reflect.Pointer, reflect.UnsafePointer:
		if a.Pointer() == e.Pointer() {
			return true, ""
		}
		res, str := equal(reflect.Indirect(a), reflect.Indirect(e))
		if !res && str == "" {
			str = fmt.Sprintf("actual value '%v' differs from expected '%v'", reflect.Indirect(a), reflect.Indirect(e))
		}
		return res, str
	case reflect.Array:
		// u and v have the same type so they have the same length
		vl := a.Len()
		if vl == 0 {
			return true, ""
		}
		for i := 0; i < vl; i++ {
			res, str := equal(a.Index(i), e.Index(i))
			if !res {
				if str == "" {
					str = fmt.Sprintf("actual value '%v' differs from expected '%v'",
						a.Index(i), e.Index(i))
				}
				return false, fmt.Sprintf("array elem %d: %s", i, str)
			}
		}
		return true, ""
	case reflect.Struct:
		// u and v have the same type so they have the same fields
		nf := a.NumField()
		for i := 0; i < nf; i++ {
			res, str := equal(a.Field(i), e.Field(i))
			if !res {
				if str == "" {
					str = fmt.Sprintf("actual value '%v' differs from expected '%v'",
						a.Field(i), e.Field(i))
				}
				return false, fmt.Sprintf("struct field '%s': %s", a.Type().Field(i).Name, str)
			}
		}
		return true, ""
	case reflect.Map:
		if a.Pointer() == e.Pointer() {
			return true, ""
		}
		keys := a.MapKeys()
		if len(keys) != len(e.MapKeys()) {
			return false, "map lengths differ"
		}
		for _, k := range keys {
			res, str := equal(a.MapIndex(k), e.MapIndex(k))
			if !res {
				if str == "" {
					str = fmt.Sprintf("actual value '%v' differs from expected '%v'",
						a.MapIndex(k), e.MapIndex(k))
				}
				return false, fmt.Sprintf("map value for key '%v': %s", k, str)
			}
		}
		return true, ""
	case reflect.Func:
		return a.Pointer() == e.Pointer(), ""
		// function can be equal only to itself
	case reflect.Slice:
		if a.Pointer() == e.Pointer() {
			return true, ""
		}
		vl := a.Len()
		if vl != e.Len() {
			return false, "slice lengths differ"
		}
		if vl == 0 {
			return true, ""
		}
		for i := 0; i < vl; i++ {
			res, str := equal(a.Index(i), e.Index(i))
			if !res {
				if str == "" {
					str = fmt.Sprintf("actual value '%v' differs from expected '%v'",
						a.Index(i), e.Index(i))
				}
				return false, fmt.Sprintf("slice elem %d: %s", i, str)
			}
		}
		return true, ""
	}
	return false, "invalid variable Kind" // should never happen
}
