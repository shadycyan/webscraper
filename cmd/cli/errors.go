package main

import "fmt"

type statusCodeError struct {
	statusCode int
	statusText string
}

func (e *statusCodeError) Error() string {
	return fmt.Sprintf("received error status code: %d %s", e.statusCode, e.statusText)
}

type contentTypeError struct {
	contentType  string
	expectedType string
}

func (e *contentTypeError) Error() string {
	return fmt.Sprintf("invalid content type: %s, expected %s", e.contentType, e.expectedType)
}
