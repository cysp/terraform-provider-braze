package testing

import (
	"fmt"
	"net/http"
)

var errNotFound = newStatusCodeError(http.StatusNotFound)

type statusCodeError struct {
	StatusCode int
}

var _ error = (*statusCodeError)(nil)

func newStatusCodeError(statusCode int) statusCodeError {
	return statusCodeError{
		StatusCode: statusCode,
	}
}

func (e statusCodeError) Error() string {
	return fmt.Sprintf("error: %d", e.StatusCode)
}
