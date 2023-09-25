package build

import "github.com/solidDoWant/distrobuilder/internal/utils"

type ITargetTripletBuilder interface {
	SetTargetTriplet(*utils.Triplet)
	GetTargetTriplet() *utils.Triplet
}

type TargetTripletBuilder struct {
	Triplet *utils.Triplet
}

func (ttb *TargetTripletBuilder) SetTargetTriplet(triplet *utils.Triplet) {
	ttb.Triplet = triplet
}

func (ttb *TargetTripletBuilder) GetTargetTriplet() *utils.Triplet {
	return ttb.Triplet
}
