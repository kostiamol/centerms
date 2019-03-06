package errors

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

// APIError .
type APIError struct {
	Code    string `json:"status"`
	Message string `json:"code"`
}

// Error .
func (err APIError) Error() string {
	return err.Message
}

// NewBadJwtError .
func NewBadJwtError(msg string) APIError {
	logrus.Debugf("NewBadJwtError: %s", msg)
	return APIError{
		Code:    ErrBadJwt,
		Message: "Unauthorized",
	}
}

// NewAuthorizationError .
func NewAuthorizationError() APIError {
	return APIError{
		Code:    ErrAuth,
		Message: "Unauthorized",
	}
}

// NewNotFoundError .
func NewNotFoundError() APIError {
	return APIError{
		Code:    ErrNotFound,
		Message: "Not Found",
	}
}

// NewValidationError .
func NewValidationError(err url.Values) ValidationError {
	return ValidationError{
		APIError: APIError{
			Code:    ErrBadParam,
			Message: "Bad Request",
		},
		Errors: err,
	}
}

// ValidationError validation errors
type ValidationError struct {
	APIError
	GlobalMessage []string
	Errors        url.Values
}

// Error .
func (err ValidationError) Error() string {
	return err.Message
}
