package errwrapper_test

import (
	"errors"
	"github.com/semirm-dev/iopipe/errwrapper"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWrap_ByOne(t *testing.T) {
	var errWrapped error

	err := errors.New("test1")
	err2 := errors.New("test2")

	errWrapped = errwrapper.Wrap(errWrapped, err)
	errWrapped = errwrapper.Wrap(errWrapped, err2)

	assert.Equal(t, errors.New("test1:test2"), errWrapped)
}

func TestWrap_Multiple(t *testing.T) {
	var errWrapped error

	err := errors.New("test1")
	err2 := errors.New("test2")

	errWrapped = errwrapper.Wrap(errWrapped, err, err2)

	assert.Equal(t, errors.New("test1:test2"), errWrapped)
}

func TestWrap_AllNil(t *testing.T) {
	var errWrapped error

	errWrapped = errwrapper.Wrap(errWrapped, nil)

	assert.NoError(t, errWrapped)
}

func TestWrap_PartiallyNil(t *testing.T) {
	var errWrapped error

	errWrapped = errwrapper.Wrap(errWrapped, errors.New("test1"), nil, errors.New("test2"))

	assert.Equal(t, errors.New("test1:test2"), errWrapped)
}
