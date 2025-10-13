package errors

import (
	"errors"
	"fmt"
)

// 预定义的错误类型
var (
	ErrInvalidInput           = errors.New("invalid input")
	ErrUpstreamServiceFailed  = errors.New("upstream service failed")
	ErrAuthenticationRequired = errors.New("authentication required")
	ErrRateLimited            = errors.New("rate limited")
	ErrNotFound               = errors.New("not found")
	ErrInternalServer         = errors.New("internal server error")
)

// UpstreamError 包含上游服务错误的详细信息
type UpstreamError struct {
	StatusCode int
	Message    string
	Err        error
}

func (e *UpstreamError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s (status: %d): %v", e.Message, e.StatusCode, e.Err)
	}
	return fmt.Sprintf("%s (status: %d)", e.Message, e.StatusCode)
}

func (e *UpstreamError) Unwrap() error {
	return e.Err
}

// NewUpstreamError 创建一个新的上游错误
func NewUpstreamError(statusCode int, message string, err error) *UpstreamError {
	return &UpstreamError{
		StatusCode: statusCode,
		Message:    message,
		Err:        err,
	}
}