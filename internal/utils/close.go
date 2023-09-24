package utils

import (
	"github.com/gravitational/trace"
)

type Closable interface {
	Close() error
}

// Utility function to make `defer c.Close()` logic a little easier to read
func Close(c Closable, callerErrRef *error) {
	if c == nil {
		return
	}

	// This doens't cover every case but covers the most common one
	if IsNil(c) {
		return
	}

	closeErr := c.Close()
	if closeErr == nil || callerErrRef != nil {
		return
	}

	*callerErrRef = trace.Wrap(closeErr, "failed to close resource")
}

func CloseChannel(c Closable, errChannel chan<- error) {
	var err error
	Close(c, &err)

	if err != nil {
		errChannel <- err
	}
}
