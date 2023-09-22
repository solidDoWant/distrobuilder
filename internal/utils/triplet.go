package utils

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/gravitational/trace"
)

type TripletOS struct {
	Kernel string
	LibC   string
}

func NewTripletOS(kernel, libc string) *TripletOS {
	return &TripletOS{
		Kernel: kernel,
		LibC:   libc,
	}
}

func ParseTripletOS(val string) (*TripletOS, error) {
	if val == "" {
		return nil, trace.Errorf("provided value is empty")
	}

	kernel, libc, _ := strings.Cut(val, "-")
	if kernel == "" {
		return nil, trace.Errorf("kernel value is unset")
	}

	return NewTripletOS(kernel, libc), nil
}

func (tos *TripletOS) String() string {
	if tos.LibC == "" {
		return tos.Kernel
	}

	return fmt.Sprintf("%s-%s", tos.Kernel, tos.LibC)
}

type Triplet struct {
	Machine string
	Vendor  string
	*TripletOS
}

func NewTriplet(machine, vendor string, os *TripletOS) *Triplet {
	return &Triplet{
		Machine:   machine,
		Vendor:    vendor,
		TripletOS: os,
	}
}

func ParseTriplet(val string) (*Triplet, error) {
	if val == "" {
		return nil, trace.Errorf("provided value is empty")
	}

	machine, secondHalf, _ := strings.Cut(val, "-")
	if secondHalf == "" {
		return nil, trace.Errorf("provided triplet string %q does not contain a vendor and/or OS", val)
	}

	secondHalfLeft, secondHalfRight, _ := strings.Cut(secondHalf, "-")

	// This can take on many for the OS, but the only OS supported by this tool is Linux anyway
	var tripletOSValue string
	vendor := ""
	if strings.ToLower(secondHalfLeft) == "linux" {
		tripletOSValue = secondHalf
	} else {
		vendor = secondHalfLeft
		tripletOSValue = secondHalfRight
	}

	tos, err := ParseTripletOS(tripletOSValue)
	if err != nil {
		return nil, trace.Wrap(err, "failed to parse triplet OS for %q", val)
	}

	return NewTriplet(machine, vendor, tos), nil
}

func (t *Triplet) String() string {
	builtString := ""

	builtString += t.Machine

	if t.Vendor != "" {
		builtString += "-"
		builtString += t.Vendor
	}

	builtString += "-"
	builtString += t.TripletOS.String()

	return builtString
}

func (t *Triplet) GetDynamicLoaderName() string {
	return fmt.Sprintf("ld-%s-%s.so.1", t.LibC, t.Machine)
}

func GetTripletMachineValue() string {
	switch runtime.GOARCH {
	case "386":
		return "x86"
	case "amd64":
		return "x86_64"
	default:
		return runtime.GOARCH
	}
}
