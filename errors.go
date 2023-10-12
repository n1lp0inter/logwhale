package logwhale

import (
	"fmt"
	"strings"
)

// ErrorState defines a set of concrete error states that can be encountered by the LogManager.
type ErrorState int

// String() returns a string representation of the ErrorState.
func (es ErrorState) String() string {
	switch es {
	case ErrorStateUnknown:
		return "Unknown"
	case ErrorStateEndOfStream:
		return "End of stream"
	case ErrorStateFileNotExist:
		return "File does not exist"
	case ErrorStateFileRemoved:
		return "File removed"
	case ErrorStateCancelled:
		return "Operation cancelled"
	case ErrorStateFSWatcher:
		return "FS watch process error"
	case ErrorStateFilePath:
		return "File path error"
	case ErrorStateFileIO:
		return "File IO error"
	case ErrorStateInternal:
		return "Internal exception"
	default:
		return "Unknown"
	}
}

const (
	ErrorStateUnknown ErrorState = iota
	ErrorStateCancelled
	ErrorStateFSWatcher
	ErrorStateFilePath
	ErrorStateFileNotExist
	ErrorStateFileRemoved
	ErrorStateFileIO
	ErrorStateEndOfStream
	ErrorStateInternal
)

type LogWhaleError struct {
	State ErrorState
	Msg   string
	Cause error
}

func NewLogWhaleError(state ErrorState, msg string, cause error) *LogWhaleError {
	return &LogWhaleError{
		State: state,
		Msg:   msg,
		Cause: cause,
	}
}

// Error satisfies the error interface.
func (e *LogWhaleError) Error() string {
	es := strings.Builder{}
	es.WriteString(fmt.Sprintf("state: %s", e.State))

	if len(e.Msg) != 0 {
		es.WriteString(fmt.Sprintf(" msg: %s", e.Msg))
	}

	if e.Cause != nil {
		es.WriteString(fmt.Sprintf(" cause: %s", e.Cause))

	}
	return es.String()
}

// Unwrap satisfies the Wrapper interface. It allows the
// LogWhaleError to work with and errors.As.
func (e *LogWhaleError) Unwrap() error {
	return e.Cause
}
