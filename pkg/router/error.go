package router

import (
	"fmt"
	"net/http"
)

type RouterError struct {
	StatusCode int    `json:"status"`
	Message    string `json:"message"`
}

// Implement the error interface for CustomError
func (e *RouterError) Error() string {
	return fmt.Sprintf("Status Code: %d - Message: %s", e.StatusCode, e.Message)
}

func NotFoundError(msg string) *RouterError {
	return &RouterError{
		StatusCode: http.StatusNotFound,
		Message:    msg,
	}
}

func CustomError(status int, msg string) *RouterError {
	return &RouterError{
		StatusCode: status,
		Message:    msg,
	}
}

func BadReqError(msg string) *RouterError {
	return &RouterError{
		StatusCode: http.StatusBadRequest,
		Message:    msg,
	}
}
