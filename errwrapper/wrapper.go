package errwrapper

import (
	"errors"
	"fmt"
	"strings"
)

const separator = ":"

// Wrap will append errs to existing err and return new error with all previous errors
func Wrap(err error, errs ...error) error {
	var errMessages []string
	for _, e := range errs {
		if e == nil {
			continue
		}
		errMessages = append(errMessages, e.Error())
	}

	errsFormatted := strings.Join(errMessages, separator)

	if len(errsFormatted) == 0 {
		return nil
	}

	if err == nil {
		err = errors.New(errsFormatted)
	} else {
		err = fmt.Errorf("%s%s%s", err, separator, errsFormatted)
	}

	return err
}
