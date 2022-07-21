package e

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

var ErrNotFound = NewNotFound("not_found")

type Error interface {
	Code() int
	Detail() string
	Error() string
}

type httpError struct {
	detail string
	code   int
}

func (e httpError) Error() string {
	return fmt.Sprintf(`code: %d, detail: '%s'`, e.code, e.detail)
}

func (e httpError) Code() int {
	return e.code
}

func (e httpError) Detail() string {
	return e.detail
}

func NewInternal(detail string) Error {
	return httpError{
		detail: detail,
		code:   http.StatusInternalServerError,
	}
}

func NewNotFound(detail string) Error {
	return httpError{
		detail: detail,
		code:   http.StatusNotFound,
	}
}

func HTTPError(err error) *echo.HTTPError {
	if he, ok := err.(httpError); ok {
		return echo.NewHTTPError(he.code, he.detail)
	}

	return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
}
