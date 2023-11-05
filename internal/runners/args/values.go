package args

import "strconv"

type IValue interface {
	GetValue() string
}

type StringValue string

func (sv StringValue) GetValue() string {
	return string(sv)
}

type CMakeBoolValue struct {
	IsOn     bool
	IsForced bool // Some vars (but not all can take a "forced" value)
}

func (bv CMakeBoolValue) GetValue() string {
	if !bv.IsOn {
		return "OFF"
	}

	if bv.IsForced {
		return "FORCED_ON"
	}

	return "ON"
}

func OffValue() CMakeBoolValue      { return CMakeBoolValue{} }
func OnValue() CMakeBoolValue       { return CMakeBoolValue{IsOn: true} }
func ForcedOnValue() CMakeBoolValue { return CMakeBoolValue{IsOn: true, IsForced: true} }

type emptyValue struct{}

func (*emptyValue) GetValue() string {
	return ""
}

var EmptyValue = &emptyValue{}

type BoolValue bool

func (bv BoolValue) GetValue() string {
	return strconv.FormatBool(bool(bv))
}

func FalseValue() BoolValue { return BoolValue(false) }
func TrueValue() BoolValue  { return BoolValue(true) }
