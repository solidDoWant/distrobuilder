package utils

import (
	"fmt"
	"strings"

	"github.com/gravitational/trace"
)

type TripletOS struct {
	Kernel string
	LibC   string
}

func ParseTripletOS(val string) (*TripletOS, error) {
	if val == "" {
		return nil, trace.Errorf("provided value is empty")
	}

	kernel, libc, _ := strings.Cut(val, "-")

	return &TripletOS{
		Kernel: kernel,
		LibC:   libc,
	}, nil
}

func (tos *TripletOS) AsString() (string, error) {
	if tos.Kernel == "" {
		return "", trace.Errorf("kernel value is unset")
	}

	if tos.LibC == "" {
		return tos.Kernel, nil
	}

	return fmt.Sprintf("%s-%s", tos.Kernel, tos.LibC), nil
}

type Triplet struct {
	Machine string
	Vendor  string
	*TripletOS
}

func ParseTriplet(val string) (*Triplet, error) {
	if val == "" {
		return nil, trace.Errorf("provided value is empty")
	}

	machine, secondHalf, _ := strings.Cut(val, "-")
	if secondHalf == "" {
		return nil, trace.Errorf("provided triplet string %q does not contain a vendor and/or OS", val)
	}

	triplet := &Triplet{
		Machine: machine,
	}

	secondHalfLeft, secondHalfRight, _ := strings.Cut(secondHalf, "-")

	// This can take on many for the OS, but the only OS supported by this tool is Linux anyway
	var tripletOSValue string
	if strings.ToLower(secondHalfLeft) == "linux" {
		tripletOSValue = secondHalf
	} else {
		triplet.Vendor = secondHalfLeft
		tripletOSValue = secondHalfRight
	}

	tos, err := ParseTripletOS(tripletOSValue)
	if err != nil {
		return nil, trace.Wrap(err, "failed to parse triplet OS for %q", val)
	}

	triplet.TripletOS = tos

	return triplet, nil
}

func (t *Triplet) AsString() (string, error) {
	builtString := ""

	builtString += t.Machine

	if t.Vendor != "" {
		builtString += "-"
		builtString += t.Vendor
	}

	osString, err := t.TripletOS.AsString()
	if err != nil {
		return "", trace.Wrap(err, "failed to convert triplet OS into string")
	}
	builtString += "-"
	builtString += osString

	return builtString, nil
}
