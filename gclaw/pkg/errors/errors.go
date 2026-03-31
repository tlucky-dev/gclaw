package errors

import "fmt"

// ErrorCode 错误码类型
type ErrorCode string

const (
	ErrUnknown         ErrorCode = "unknown_error"
	ErrConfig          ErrorCode = "config_error"
	ErrProvider        ErrorCode = "provider_error"
	ErrTool            ErrorCode = "tool_error"
	ErrMemory          ErrorCode = "memory_error"
	ErrEngine          ErrorCode = "engine_error"
	ErrInvalidRequest  ErrorCode = "invalid_request"
)

// GCLawError 自定义错误类型
type GCLawError struct {
	Code    ErrorCode
	Message string
	Cause   error
}

func (e *GCLawError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *GCLawError) Unwrap() error {
	return e.Cause
}

// NewError 创建新错误
func NewError(code ErrorCode, message string) *GCLawError {
	return &GCLawError{
		Code:    code,
		Message: message,
	}
}

// WrapError 包装现有错误
func WrapError(code ErrorCode, message string, cause error) *GCLawError {
	return &GCLawError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}
