package glfw

import "fmt"

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
