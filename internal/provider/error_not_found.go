package provider

import "errors"

type NotFoundError struct {
	message string
}

func (e *NotFoundError) Error() string { return e.message }

// isNotFound reports whether err, or an error it wraps, is a NotFoundError.
func isNotFound(err error) bool {
	var notFound *NotFoundError
	return errors.As(err, &notFound)
}
