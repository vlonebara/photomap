package service

import "errors"

var (
	ErrNotFound        = errors.New("photo or series not found")
	ErrUnauthorized    = errors.New("authentication required")
	ErrForbidden       = errors.New("access denied")
	ErrCredentials     = errors.New("invalid email or password")
	ErrEmailExists     = errors.New("email already registered")
	ErrAlreadyReported = errors.New("photo already reported")
	ErrConflict        = errors.New("operation is not allowed in the current state")
	ErrImageTooLarge   = errors.New("image exceeds the upload limit")
)

// field errors shared by registration, uploads and moderation
type ValidationError struct {
	Message string
	Fields  map[string]string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// creating an error for one invalid field
func invalid(field, message string) error {
	return &ValidationError{Message: message, Fields: map[string]string{field: message}}
}
