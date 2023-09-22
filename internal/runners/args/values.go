package args

type IValue interface {
	GetValue() string
}

type StringValue string

func (sv StringValue) GetValue() string {
	return string(sv)
}

type BoolValue struct {
	IsOn     bool
	IsForced bool // Some vars (but not all can take a "forced" value)
}

func (bv BoolValue) GetValue() string {
	if !bv.IsOn {
		return "OFF"
	}

	if bv.IsForced {
		return "FORCED_ON"
	}

	return "ON"
}

func OffValue() BoolValue      { return BoolValue{} }
func OnValue() BoolValue       { return BoolValue{IsOn: true} }
func ForcedOnValue() BoolValue { return BoolValue{IsOn: true, IsForced: true} }

type emptyValue struct{}

func (*emptyValue) GetValue() string {
	return ""
}

var EmptyValue = &emptyValue{}
