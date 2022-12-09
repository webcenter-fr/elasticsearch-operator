package helper

import (
	"reflect"
	"strings"
)

// Generic function to remove item from a slice
func DeleteItemFromSlice(x any, index int) any {
	xValue := reflect.ValueOf(x)
	xType := xValue.Type()
	if xType.Kind() != reflect.Slice {
		panic("First parameter must be a slice")
	}

	expectedSlice := reflect.MakeSlice(reflect.SliceOf(xType.Elem()), 0, xValue.Len()-1)

	for i := 0; i < xValue.Len(); i++ {
		if i != index {
			expectedSlice = reflect.Append(expectedSlice, xValue.Index(i))
		}
	}

	return expectedSlice.Interface()
}

func StringToSlice(value, separator string) (result []string) {
	if value == "" {
		return []string{}
	}
	result = strings.Split(value, separator)
	for i, s := range result {
		result[i] = strings.TrimSpace(s)
	}
	return result
}
