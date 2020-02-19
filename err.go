package writerset

import (
	"errors"
	"io"
)

var errInvalidErrPartialWrite = errors.New("invalid ErrPartialWrite: missing underlying error")

// ErrPartialWrite encapsulates an error from a WriterSet.
type ErrPartialWrite struct {
	Writer          io.Writer
	Err             error
	Expected, Wrote int
}

// Error returns the error string from the underlying error.
func (e ErrPartialWrite) Error() string {
	if e.Err == nil {
		return errInvalidErrPartialWrite.Error()
	}
	return e.Err.Error()
}

// Unwrap returns the underlying write error.
func (e ErrPartialWrite) Unwrap() error {
	if e.Err == nil {
		return errInvalidErrPartialWrite
	}
	return e.Err
}
