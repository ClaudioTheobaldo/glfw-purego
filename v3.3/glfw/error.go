package glfw

import (
	"fmt"
	"sync"
)

// errorCb is the user-registered error callback (see SetErrorCallback).
// Errors are still returned in-band as *Error from the failing function;
// the callback is an additional notification mechanism that mirrors the
// GLFW C API and upstream go-gl/glfw.
var (
	errorMu sync.Mutex
	errorCb func(code ErrorCode, desc string)
)

// SetErrorCallback registers cb to be invoked whenever a GLFW operation
// reports an error.  Returns the previous callback, if any.  Pass nil to
// unregister.
//
// The callback receives the same (code, desc) pair embedded in the
// *Error value returned by the failing function — registering one is
// optional but useful when you want a single point of error logging
// across the whole library.
func SetErrorCallback(cb func(code ErrorCode, desc string)) func(code ErrorCode, desc string) {
	errorMu.Lock()
	defer errorMu.Unlock()
	prev := errorCb
	errorCb = cb
	return prev
}

// emitError reports an error to the registered callback, if any.  Safe
// to call without holding the caller's locks.  Internal helpers should
// invoke this whenever they construct an *Error to return.
func emitError(code ErrorCode, desc string) {
	errorMu.Lock()
	cb := errorCb
	errorMu.Unlock()
	if cb != nil {
		cb(code, desc)
	}
}

// ErrorCode represents a GLFW error code.
type ErrorCode int

const (
	NotInitialized     ErrorCode = 0x00010001
	NoCurrentContext   ErrorCode = 0x00010002
	InvalidEnum        ErrorCode = 0x00010003
	InvalidValue       ErrorCode = 0x00010004
	OutOfMemory        ErrorCode = 0x00010005
	APIUnavailable     ErrorCode = 0x00010006
	VersionUnavailable ErrorCode = 0x00010007
	PlatformError      ErrorCode = 0x00010008
	FormatUnavailable  ErrorCode = 0x00010009
	NoWindowContext    ErrorCode = 0x0001000A
)

func (e ErrorCode) String() string {
	switch e {
	case NotInitialized:
		return "NotInitialized"
	case NoCurrentContext:
		return "NoCurrentContext"
	case InvalidEnum:
		return "InvalidEnum"
	case InvalidValue:
		return "InvalidValue"
	case OutOfMemory:
		return "OutOfMemory"
	case APIUnavailable:
		return "APIUnavailable"
	case VersionUnavailable:
		return "VersionUnavailable"
	case PlatformError:
		return "PlatformError"
	case FormatUnavailable:
		return "FormatUnavailable"
	case NoWindowContext:
		return "NoWindowContext"
	default:
		return fmt.Sprintf("ErrorCode(%d)", int(e))
	}
}

// Error is a GLFW error with a code and a human-readable description.
type Error struct {
	Code ErrorCode
	Desc string
}

func (e *Error) Error() string {
	return fmt.Sprintf("glfw: %s: %s", e.Code, e.Desc)
}

func newError(code ErrorCode, desc string) *Error {
	return &Error{Code: code, Desc: desc}
}
