package api

import (
	"net/url"
)

const (
	errService    = "ERR_SERVICE"
	errNotFound   = "ERR_NOT_FOUND"
	errBadRequest = "ERR_BAD_REQUEST"
	errBadParam   = "ERR_BAD_PARAM"
	errAuth       = "ERR_NOT_AUTHORIZED"
	errBadJWT     = "ERR_BAD_JWT"
)

type apiError struct {
	Code    string `json:"status"`
	Message string `json:"code"`
}

// Error returns error as a string.
func (e apiError) Error() string {
	return e.Message
}

func newBadJWTError(msg string) apiError {
	return apiError{
		Code:    errBadJWT,
		Message: "Unauthorized",
	}
}

func newAuthorizationError() apiError { // nolint
	return apiError{
		Code:    errAuth,
		Message: "Unauthorized",
	}
}

func newNotFoundError() apiError { // nolint
	return apiError{
		Code:    errNotFound,
		Message: "Not found",
	}
}

func newValidationError(err url.Values) validationError { // nolint
	return validationError{
		apiError: apiError{
			Code:    errBadParam,
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

// Error returns error as a string.
func (e validationError) Error() string {
	return e.Message
}
