package chatkit

import (
	"fmt"

	"go.openai.org/lib/go/helpers"
)

type HTTPError struct {
	Code    int
	Message string
	Caller  string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("%d %s", e.Code, e.Message)
}

func NewHTTPError(code int, msg string) error {
	return &HTTPError{Code: code, Message: msg, Caller: helpers.ThisCaller()}
}
