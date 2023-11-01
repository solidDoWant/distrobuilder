package utils

import (
	"slices"

	"github.com/elliotchance/pie/v2"
)

func DedupeReduce[T comparable](s ...[]T) []T {
	return Dedupe(pie.Flat(s))
}

// This function preserves order of values in `s`.
func Dedupe[T comparable](s []T) []T {
	result := make([]T, 0, len(s))
	for _, value := range s {
		if slices.Contains(result, value) {
			continue
		}

		result = append(result, value)
	}

	return slices.Clip(result)
}

func FilterNil[T any](s []T) []T {
	return pie.Filter(s, func(value T) bool { return !IsNil(s) })
}

// Updates the values of src, beginning at startPosition, with the newValues.
// Syntactic sugar of this would be something like:
// src[startPosition:startPosition+len(newValues)] = newValues
func UpdateSubset[T any](src *[]T, startPosition int, newValues *[]T) {
	for offset := 0; offset < len(*newValues); offset++ {
		(*src)[startPosition+offset] = (*newValues)[offset]
	}
}
