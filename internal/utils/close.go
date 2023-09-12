package utils

import "github.com/gravitational/trace"

type Closable interface {
	Close() error
}

// Utility function to make `defer c.Close()` logic a little easier to read
func Close(c Closable, callerErrRef *error) {
	if c == nil {
		return
	}

	closeErr := c.Close()
	if closeErr == nil || callerErrRef != nil {
		return
	}

	*callerErrRef = trace.Wrap(closeErr, "failed to close resource")
}
