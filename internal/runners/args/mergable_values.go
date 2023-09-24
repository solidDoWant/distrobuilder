package args

import (
	"fmt"
	"strings"

	"github.com/elliotchance/pie/v2"
	"github.com/gravitational/trace"
	"github.com/solidDoWant/distrobuilder/internal/utils"
)

type IMergableValue interface {
	IValue
	Merge(other IMergableValue) (IMergableValue, error) // Return an error if the two types are not mergable, or they failed to merge due to some other reason
}

type SeparatorValue struct {
	Values    []IValue
	Separator string
}

// This doesn't have to be called - it's basically syntactic sugar
// If the provided value is not a Value instance then it is
// converted to a string and added via a SimpleValue instance
func SeparatorValues[T string | rune](separator T, values ...any) *SeparatorValue {
	convertedValues := make([]IValue, 0, len(values))
	for _, value := range values {
		convertedValue, ok := value.(IValue)
		if ok {
			convertedValues = append(convertedValues, convertedValue)
		}

		convertedValues = append(convertedValues, StringValue(fmt.Sprintf("%v", value)))
	}

	return &SeparatorValue{
		Values:    convertedValues,
		Separator: string(separator),
	}
}

func (sv *SeparatorValue) GetValue() string {
	values := pie.Map(sv.Values, func(value IValue) string {
		return value.GetValue()
	})
	return strings.Join(values, sv.Separator)
}

func (sv *SeparatorValue) Merge(otherMergable IMergableValue) (IMergableValue, error) {
	otherSeparatorVar, ok := otherMergable.(*SeparatorValue)
	if !ok {
		return nil, trace.BadParameter("provided mergable value is of type %T but needed %T", otherMergable, sv)
	}

	if sv.Separator != otherSeparatorVar.Separator {
		return nil, trace.BadParameter("provided separator values do not match, %q != %q", sv.Separator, otherSeparatorVar.Separator)
	}

	return &SeparatorValue{
		Values:    utils.DedupeReduce(sv.Values, otherSeparatorVar.Values),
		Separator: sv.Separator,
	}, nil
}

func MergeMap[T comparable](argMaps ...map[T]IValue) (map[T]IValue, error) {
	var err error
	reduceFunc := func(map1, map2 map[T]IValue) map[T]IValue {
		if err != nil {
			// Skip doing anything and return as quickly as possible if an error already occured
			return nil
		}

		keys := pie.Unique(append(pie.Keys(map1), pie.Keys(map2)...))
		mergedMap := make(map[T]IValue, len(keys))
		for _, key := range keys {
			val1, ok1 := map1[key]
			val2, ok2 := map2[key]

			if ok1 && ok2 {
				mergableVal1, ok1 := val1.(IMergableValue)
				if !ok1 {
					err = trace.Errorf("attempted to merge unmergable value %#v with %#v for key %v", val1, val2, key)
					return nil
				}

				mergableVal2, ok2 := val2.(IMergableValue)
				if !ok2 {
					err = trace.Errorf("attempted to merge unmergable value %#v with %#v for key %v", val2, val1, key)
					return nil
				}

				mergedMap[key], err = mergableVal1.Merge(mergableVal2)
				if err != nil {
					err = trace.Wrap(err, "failed to merge %#v into %#v for key %v", val2, val1, key)
					return nil
				}

				continue
			}

			if ok1 {
				mergedMap[key] = val1
				continue
			}

			if ok2 {
				mergedMap[key] = val2
				continue
			}

			err = trace.Errorf("attempted to merge two unmergable values %#v and %#v for key %v", val1, val2, key)
			return nil
		}

		return mergedMap
	}

	mergedMap := pie.Reduce(argMaps, reduceFunc)

	if err != nil {
		return nil, err
	}

	return mergedMap, nil
}
