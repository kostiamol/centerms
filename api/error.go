package api

import (
	"net/url"

	"github.com/Sirupsen/logrus"
)

const (
	ErrService    = "ERR_SERVICE"
	ErrNotFound   = "ERR_NOT_FOUND"
	ErrBadRequest = "ERR_BAD_REQUEST"
	ErrBadParam   = "ERR_BAD_PARAM"
	ErrAuth       = "ERR_NOT_AUTHORIZED"
	ErrBadJwt     = "ERR_BAD_JWT"
)

type apiError struct {
	Code    string `json:"status"`
	Message string `json:"code"`
}

// Error() returns error as a string.
func (e apiError) Error() string {
	return e.Message
}

func newBadJWTError(msg string) apiError {
	logrus.Debugf("newBadJWTError: %s", msg)
	return apiError{
		Code:    ErrBadJwt,
		Message: "Unauthorized",
	}
}

func newAuthorizationError() apiError {
	return apiError{
		Code:    ErrAuth,
		Message: "Unauthorized",
	}
}

func newNotFoundError() apiError {
	return apiError{
		Code:    ErrNotFound,
		Message: "Not found",
	}
}

func newValidationError(err url.Values) validationError {
	return validationError{
		apiError: apiError{
			Code:    ErrBadParam,
			Message: "Bad request",
		},
		Errors: err,
	}
}

type validationError struct {
	apiError
	GlobalMessage []string
	Errors        url.Values
}

func (e validationError) Error() string {
	return e.Message
}
