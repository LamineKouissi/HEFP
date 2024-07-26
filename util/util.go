package util

import (
	"errors"
	"reflect"
)

func IsStructEmpty(str any) (bool, error) {
	v := reflect.ValueOf(str)

	// If it's a pointer, get the element it points to
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// Check if the input is a struct
	if v.Kind() != reflect.Struct {
		return false, errors.New("input type is not a struct")
	}

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)

		switch field.Kind() {
		case reflect.Struct:
			// Recursively check nested structs
			rslt, err := IsStructEmpty(field.Interface())
			if err != nil {
				return rslt, err
			}
			if !rslt {
				return false, nil
			}
		case reflect.Slice, reflect.Map:
			// Check if slice or map is empty
			if field.Len() > 0 {
				return false, nil
			}
		case reflect.Func:
			// Check if function is nil
			if !field.IsNil() {
				return false, nil
			}
		default:
			// For other types, compare with zero value
			if !reflect.DeepEqual(field.Interface(), reflect.Zero(field.Type()).Interface()) {
				return false, nil
			}
		}
	}

	return true, nil
}
