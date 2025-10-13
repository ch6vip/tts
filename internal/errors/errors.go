package errors

import "errors"

var (
	ErrInvalidInput         = errors.New("invalid input")
	ErrUpstreamServiceFailed = errors.New("upstream service failed")
	ErrNotFound             = errors.New("not found")
	ErrRateLimited          = errors.New("rate limited")
)