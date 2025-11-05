package willys

import (
	"fmt"
	"net/http"
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return e.Message
}

func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{Field: field, Message: message}
}

type AuthenticationError struct {
	Message string
	Cause   error
}

func (e *AuthenticationError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *AuthenticationError) Unwrap() error {
	return e.Cause
}

func NewAuthenticationError(message string, cause error) *AuthenticationError {
	return &AuthenticationError{Message: message, Cause: cause}
}

type APIError struct {
	StatusCode int
	Message    string
	Endpoint   string
	Cause      error
}

func (e *APIError) Error() string {
	msg := fmt.Sprintf("%s (%d)", http.StatusText(e.StatusCode), e.StatusCode)
	if e.Endpoint != "" {
		msg = fmt.Sprintf("%s at %s", msg, e.Endpoint)
	}
	if e.Message != "" {
		msg = fmt.Sprintf("%s: %s", msg, e.Message)
	}
	if e.Cause != nil {
		msg = fmt.Sprintf("%s: %v", msg, e.Cause)
	}
	return msg
}

func (e *APIError) Unwrap() error {
	return e.Cause
}

func NewAPIError(statusCode int, endpoint, message string, cause error) *APIError {
	return &APIError{
		StatusCode: statusCode,
		Endpoint:   endpoint,
		Message:    message,
		Cause:      cause,
	}
}

type NotFoundError struct {
	Resource string
	ID       string
}

func (e *NotFoundError) Error() string {
	if e.ID != "" {
		return fmt.Sprintf("%s not found: %s", e.Resource, e.ID)
	}
	return fmt.Sprintf("%s not found", e.Resource)
}

func NewNotFoundError(resource, id string) *NotFoundError {
	return &NotFoundError{Resource: resource, ID: id}
}

func IsValidationError(err error) bool {
	_, ok := err.(*ValidationError)
	return ok
}

func IsAuthenticationError(err error) bool {
	_, ok := err.(*AuthenticationError)
	return ok
}

func IsAPIError(err error) bool {
	_, ok := err.(*APIError)
	return ok
}

func IsNotFoundError(err error) bool {
	_, ok := err.(*NotFoundError)
	return ok
}
