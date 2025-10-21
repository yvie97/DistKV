// Package errors provides comprehensive error handling utilities for DistKV.
// It includes error types, wrapping, and contextual error information.
package errors

import (
	"fmt"
	"runtime"
	"strings"
)

// ErrorCode represents specific error categories in the system
type ErrorCode string

const (
	// Storage errors
	ErrCodeStorageClosed    ErrorCode = "STORAGE_CLOSED"
	ErrCodeKeyNotFound      ErrorCode = "KEY_NOT_FOUND"
	ErrCodeInvalidKey       ErrorCode = "INVALID_KEY"
	ErrCodeInvalidValue     ErrorCode = "INVALID_VALUE"
	ErrCodeStorageCorrupted ErrorCode = "STORAGE_CORRUPTED"
	ErrCodeStorageFull      ErrorCode = "STORAGE_FULL"
	ErrCodeCompactionFailed ErrorCode = "COMPACTION_FAILED"

	// Network errors
	ErrCodeConnectionFailed ErrorCode = "CONNECTION_FAILED"
	ErrCodeTimeout          ErrorCode = "TIMEOUT"
	ErrCodeNodeUnreachable  ErrorCode = "NODE_UNREACHABLE"

	// Replication errors
	ErrCodeQuorumFailed      ErrorCode = "QUORUM_FAILED"
	ErrCodeInsufficientNodes ErrorCode = "INSUFFICIENT_NODES"
	ErrCodeReplicationFailed ErrorCode = "REPLICATION_FAILED"

	// Consensus errors
	ErrCodeConflict        ErrorCode = "CONFLICT"
	ErrCodeVersionMismatch ErrorCode = "VERSION_MISMATCH"

	// Configuration errors
	ErrCodeInvalidConfig ErrorCode = "INVALID_CONFIG"

	// General errors
	ErrCodeInternal ErrorCode = "INTERNAL_ERROR"
	ErrCodeUnknown  ErrorCode = "UNKNOWN_ERROR"
)

// DistKVError represents a structured error with context
type DistKVError struct {
	Code       ErrorCode              // Error classification code
	Message    string                 // Human-readable error message
	Cause      error                  // Underlying cause (if any)
	Context    map[string]interface{} // Additional context
	StackTrace string                 // Stack trace for debugging
	Retryable  bool                   // Whether the operation can be retried
}

// Error implements the error interface
func (e *DistKVError) Error() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("[%s] %s", e.Code, e.Message))

	if len(e.Context) > 0 {
		sb.WriteString(" (")
		first := true
		for key, value := range e.Context {
			if !first {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("%s=%v", key, value))
			first = false
		}
		sb.WriteString(")")
	}

	if e.Cause != nil {
		sb.WriteString(fmt.Sprintf(": %v", e.Cause))
	}

	return sb.String()
}

// Unwrap returns the underlying cause
func (e *DistKVError) Unwrap() error {
	return e.Cause
}

// IsRetryable returns whether this error indicates a retryable operation
func (e *DistKVError) IsRetryable() bool {
	return e.Retryable
}

// New creates a new DistKVError with the specified code and message
func New(code ErrorCode, message string) *DistKVError {
	return &DistKVError{
		Code:       code,
		Message:    message,
		Context:    make(map[string]interface{}),
		StackTrace: captureStackTrace(),
		Retryable:  false,
	}
}

// Wrap wraps an existing error with additional context
func Wrap(err error, code ErrorCode, message string) *DistKVError {
	if err == nil {
		return nil
	}

	// If already a DistKVError, preserve its code if not overridden
	if dkvErr, ok := err.(*DistKVError); ok {
		return &DistKVError{
			Code:       code,
			Message:    message,
			Cause:      dkvErr,
			Context:    make(map[string]interface{}),
			StackTrace: captureStackTrace(),
			Retryable:  dkvErr.Retryable,
		}
	}

	return &DistKVError{
		Code:       code,
		Message:    message,
		Cause:      err,
		Context:    make(map[string]interface{}),
		StackTrace: captureStackTrace(),
		Retryable:  false,
	}
}

// WithContext adds contextual information to an error
func (e *DistKVError) WithContext(key string, value interface{}) *DistKVError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithRetryable marks the error as retryable or not
func (e *DistKVError) WithRetryable(retryable bool) *DistKVError {
	e.Retryable = retryable
	return e
}

// captureStackTrace captures the current stack trace
func captureStackTrace() string {
	const maxDepth = 32
	var pcs [maxDepth]uintptr
	n := runtime.Callers(3, pcs[:])

	var sb strings.Builder
	frames := runtime.CallersFrames(pcs[:n])

	for {
		frame, more := frames.Next()
		sb.WriteString(fmt.Sprintf("\n\t%s:%d %s", frame.File, frame.Line, frame.Function))
		if !more {
			break
		}
	}

	return sb.String()
}

// IsCode checks if an error has a specific error code
func IsCode(err error, code ErrorCode) bool {
	if dkvErr, ok := err.(*DistKVError); ok {
		return dkvErr.Code == code
	}
	return false
}

// IsRetryableError checks if an error is retryable
func IsRetryableError(err error) bool {
	if dkvErr, ok := err.(*DistKVError); ok {
		return dkvErr.Retryable
	}
	return false
}

// Common error constructors for convenience

// NewStorageClosedError creates a storage closed error
func NewStorageClosedError() *DistKVError {
	return New(ErrCodeStorageClosed, "storage engine is closed")
}

// NewKeyNotFoundError creates a key not found error
func NewKeyNotFoundError(key string) *DistKVError {
	return New(ErrCodeKeyNotFound, "key not found").
		WithContext("key", key)
}

// NewQuorumFailedError creates a quorum failed error
func NewQuorumFailedError(needed, got int) *DistKVError {
	return New(ErrCodeQuorumFailed, "quorum requirements not met").
		WithContext("needed", needed).
		WithContext("got", got).
		WithRetryable(true)
}

// NewConnectionError creates a connection error
func NewConnectionError(address string, cause error) *DistKVError {
	return Wrap(cause, ErrCodeConnectionFailed, "failed to connect to node").
		WithContext("address", address).
		WithRetryable(true)
}

// NewTimeoutError creates a timeout error
func NewTimeoutError(operation string) *DistKVError {
	return New(ErrCodeTimeout, "operation timed out").
		WithContext("operation", operation).
		WithRetryable(true)
}

// NewInvalidConfigError creates an invalid configuration error
func NewInvalidConfigError(message string) *DistKVError {
	return New(ErrCodeInvalidConfig, message)
}
