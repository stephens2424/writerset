package writerset

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriterset(t *testing.T) {
	ws := WriterSet{}

	bufA := &bytes.Buffer{}
	bufB := &bytes.Buffer{}
	bufC := &bytes.Buffer{}
	flushableC := &flushWriter{w: bufC}

	writeNoError(t, &ws, "0")

	ws.Add(bufA)
	writeNoError(t, &ws, "1")

	ws.Add(bufB)
	writeNoError(t, &ws, "2")

	ws.Remove(bufA)
	ws.Add(flushableC)
	writeNoError(t, &ws, "3")

	assert.Equal(t, "12", bufA.String())
	assert.Equal(t, "23", bufB.String())
	assert.Equal(t, "", bufC.String()) // should be empty until flush

	ws.Flush()

	assert.Equal(t, "12", bufA.String())
	assert.Equal(t, "23", bufB.String())
	assert.Equal(t, "3", bufC.String()) // should be pupulated after flush

	assert.False(t, ws.Contains(bufA))
	assert.True(t, ws.Contains(bufB))
	assert.False(t, ws.Contains(bufC))
	assert.True(t, ws.Contains(flushableC))
}

func writeNoError(t *testing.T, w io.Writer, s string) {
	_, err := fmt.Fprint(w, s)
	require.NoError(t, err)
}

type flushWriter struct {
	buf bytes.Buffer
	w   io.Writer
}

func (fw *flushWriter) Write(b []byte) (int, error) {
	return fw.buf.Write(b)
}

func (fw *flushWriter) Flush() {
	_, _ = io.Copy(fw.w, &fw.buf)
}

func TestFail(t *testing.T) {
	ws := WriterSet{}

	bufA := &bytes.Buffer{}
	bufB := &bytes.Buffer{}
	bufC := &bytes.Buffer{}

	failB := &failWriter{failAfter: 2, w: bufB}

	chA := ws.Add(bufA)
	chB := ws.Add(failB)
	chC := ws.Add(bufC)

	n, err := fmt.Fprint(&ws, "123")
	assert.Equal(t, 3, n)
	assert.NoError(t, err)

	assert.Equal(t, "123", bufA.String())
	assert.Equal(t, "12", bufB.String())
	assert.Equal(t, "123", bufC.String())

	assert.True(t, ws.Contains(bufC))
	ws.Remove(bufC)
	assert.False(t, ws.Contains(bufC))

	// a should still be open
	errA, isClosedA := checkErrChan(chA)
	assert.NoError(t, errA)
	assert.False(t, isClosedA)

	// b should be closed with an error
	errB, isClosedB := checkErrChan(chB)
	assert.Equal(t, errB, ErrPartialWrite{
		Writer:   failB,
		Err:      errFailWriterHitLimit,
		Expected: 3,
		Wrote:    2,
	})
	assert.True(t, isClosedB)

	// c should be closed with no error
	errC, isClosedC := checkErrChan(chC)
	assert.NoError(t, errC)
	assert.True(t, isClosedC, chC)
}

func checkErrChan(ch <-chan error) (err error, isClosed bool) {
	var isOK bool

	select {
	case err, isOK = <-ch:
		if !isOK {
			isClosed = true
		} else {
			_, isClosed = checkErrChan(ch)
		}
	default:
	}

	return
}

var errFailWriterHitLimit = errors.New("failing to write beyond limit")

type failWriter struct {
	written   int
	failAfter int
	w         io.Writer
}

func (fw *failWriter) Write(b []byte) (int, error) {
	writtenAfterWrite := fw.written + len(b)

	doFail := false // used to identify the case where we are hitting the limit

	if writtenAfterWrite > fw.failAfter {
		tooFar := writtenAfterWrite - fw.failAfter
		doFail = true
		b = b[:len(b)-tooFar]
	}

	n, err := fw.w.Write(b)
	if err != nil {
		return n, err
	}

	fw.written += n
	if doFail {
		return n, errFailWriterHitLimit
	}

	return n, nil
}
