package entities

import "github.com/pkg/errors"

// Application specific errors
var (
	errFailedDecoding = errors.New("decoding has failed")
)
