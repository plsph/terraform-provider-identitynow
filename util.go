package main

import (
	"reflect"
	"strings"
)

func isEmptyValue(value interface{}) bool {
	if value == nil {
		return true
	}
	val := reflect.ValueOf(value)

	switch val.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
		return val.Len() == 0
	case reflect.Ptr, reflect.Interface:
		return val.IsNil()
	default:
		// Check zero value for other types
		zeroValue := reflect.Zero(val.Type()).Interface()
		return reflect.DeepEqual(value, zeroValue)
	}
}

func getJSONFieldName(field reflect.StructField) string {
	// Get the JSON tag value
	jsonTag := field.Tag.Get("json")
	if jsonTag == "" || jsonTag == "-" { // Handle empty or ignored fields
		return ""
	}

	// Split the tag by commas and return the first part (the field name)
	name := strings.Split(jsonTag, ",")[0]
	return name
}
