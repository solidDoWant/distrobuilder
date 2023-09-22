package utils

import "reflect"

func IsNil[T any](value T) bool {
	return reflect.TypeOf(value).Kind() == reflect.Ptr && reflect.ValueOf(value).IsNil()
}
