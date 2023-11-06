package utils

import (
	"github.com/gravitational/trace"
)

type Closable interface {
	Close() error
}

// Utility function to make `defer c.Close()` logic a little easier to read
func Close(c Closable, callerErrRef *error) {
	if IsNil(c) {
		return
	}

	ErrDefer(c.Close, callerErrRef)
	if callerErrRef != nil {
		*callerErrRef = trace.Wrap(*callerErrRef, "failed to close resource")
	}
}

func ErrDefer(deferedFunction func() error, callerErrRef *error) {
	if deferedFunction == nil {
		return
	}

	// This doens't cover every case but covers the most common one
	if IsNil(deferedFunction) {
		return
	}

	deferedError := deferedFunction()
	if deferedError == nil || callerErrRef != nil {
		return
	}

	*callerErrRef = trace.Wrap(deferedError, "defered function failed")
}

func CloseChannel(c Closable, errChannel chan<- error) {
	var err error
	Close(c, &err)

	if err != nil {
		errChannel <- err
	}
}
