// Unit tests for the errors package
package errors

import (
	stderrors "errors"
	"fmt"
	"strings"
	"testing"
)

// TestNew tests creating a new DistKVError
func TestNew(t *testing.T) {
	err := New(ErrCodeKeyNotFound, "test error message")

	if err == nil {
		t.Fatal("Expected non-nil error")
	}

	if err.Code != ErrCodeKeyNotFound {
		t.Errorf("Expected code %s, got %s", ErrCodeKeyNotFound, err.Code)
	}

	if err.Message != "test error message" {
		t.Errorf("Expected message 'test error message', got %s", err.Message)
	}

	if err.Cause != nil {
		t.Error("Expected nil cause")
	}

	if err.Context == nil {
		t.Error("Expected non-nil context map")
	}

	if err.StackTrace == "" {
		t.Error("Expected non-empty stack trace")
	}
}

// TestError tests the Error() method
func TestError(t *testing.T) {
	tests := []struct {
		name     string
		err      *DistKVError
		contains []string
	}{
		{
			name: "simple error",
			err:  New(ErrCodeInternal, "something went wrong"),
			contains: []string{
				"[INTERNAL_ERROR]",
				"something went wrong",
			},
		},
		{
			name: "error with context",
			err: New(ErrCodeKeyNotFound, "key missing").
				WithContext("key", "user:123"),
			contains: []string{
				"[KEY_NOT_FOUND]",
				"key missing",
				"key=user:123",
			},
		},
		{
			name: "error with cause",
			err: Wrap(
				fmt.Errorf("original error"),
				ErrCodeConnectionFailed,
				"failed to connect",
			),
			contains: []string{
				"[CONNECTION_FAILED]",
				"failed to connect",
				"original error",
			},
		},
		{
			name: "error with multiple context fields",
			err: New(ErrCodeQuorumFailed, "quorum not met").
				WithContext("needed", 3).
				WithContext("got", 2),
			contains: []string{
				"[QUORUM_FAILED]",
				"quorum not met",
				"needed=3",
				"got=2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.err.Error()
			for _, substr := range tt.contains {
				if !strings.Contains(errStr, substr) {
					t.Errorf("Expected error string to contain '%s', got: %s", substr, errStr)
				}
			}
		})
	}
}

// TestWrap tests wrapping errors
func TestWrap(t *testing.T) {
	t.Run("wrap standard error", func(t *testing.T) {
		originalErr := fmt.Errorf("database connection failed")
		wrapped := Wrap(originalErr, ErrCodeConnectionFailed, "connection error")

		if wrapped.Code != ErrCodeConnectionFailed {
			t.Errorf("Expected code %s, got %s", ErrCodeConnectionFailed, wrapped.Code)
		}

		if wrapped.Cause != originalErr {
			t.Error("Expected cause to be original error")
		}

		if !strings.Contains(wrapped.Error(), "database connection failed") {
			t.Error("Expected wrapped error to contain original message")
		}
	})

	t.Run("wrap DistKVError", func(t *testing.T) {
		innerErr := New(ErrCodeStorageClosed, "storage is closed").
			WithRetryable(true)
		wrapped := Wrap(innerErr, ErrCodeInternal, "operation failed")

		if wrapped.Retryable != true {
			t.Error("Expected retryable to be preserved from inner error")
		}

		if wrapped.Cause != innerErr {
			t.Error("Expected cause to be inner error")
		}
	})

	t.Run("wrap nil error", func(t *testing.T) {
		wrapped := Wrap(nil, ErrCodeInternal, "should be nil")
		if wrapped != nil {
			t.Error("Expected nil when wrapping nil error")
		}
	})
}

// TestUnwrap tests error unwrapping
func TestUnwrap(t *testing.T) {
	originalErr := fmt.Errorf("original error")
	wrapped := Wrap(originalErr, ErrCodeInternal, "wrapped")

	unwrapped := wrapped.Unwrap()
	if unwrapped != originalErr {
		t.Error("Expected Unwrap to return original error")
	}

	// Test with Is from standard library
	if !stderrors.Is(wrapped, originalErr) {
		t.Error("Expected Is to find original error")
	}
}

// TestWithContext tests adding context to errors
func TestWithContext(t *testing.T) {
	err := New(ErrCodeKeyNotFound, "key not found").
		WithContext("key", "user:123").
		WithContext("table", "users")

	if err.Context["key"] != "user:123" {
		t.Error("Expected key context to be set")
	}

	if err.Context["table"] != "users" {
		t.Error("Expected table context to be set")
	}

	if len(err.Context) != 2 {
		t.Errorf("Expected 2 context fields, got %d", len(err.Context))
	}
}

// TestWithRetryable tests marking errors as retryable
func TestWithRetryable(t *testing.T) {
	t.Run("retryable error", func(t *testing.T) {
		err := New(ErrCodeTimeout, "timeout").
			WithRetryable(true)

		if !err.IsRetryable() {
			t.Error("Expected error to be retryable")
		}

		if !IsRetryableError(err) {
			t.Error("Expected IsRetryableError to return true")
		}
	})

	t.Run("non-retryable error", func(t *testing.T) {
		err := New(ErrCodeInvalidKey, "invalid key").
			WithRetryable(false)

		if err.IsRetryable() {
			t.Error("Expected error to not be retryable")
		}

		if IsRetryableError(err) {
			t.Error("Expected IsRetryableError to return false")
		}
	})
}

// TestIsCode tests checking error codes
func TestIsCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		code     ErrorCode
		expected bool
	}{
		{
			name:     "matching code",
			err:      New(ErrCodeKeyNotFound, "not found"),
			code:     ErrCodeKeyNotFound,
			expected: true,
		},
		{
			name:     "non-matching code",
			err:      New(ErrCodeKeyNotFound, "not found"),
			code:     ErrCodeInternal,
			expected: false,
		},
		{
			name:     "standard error",
			err:      fmt.Errorf("standard error"),
			code:     ErrCodeInternal,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsCode(tt.err, tt.code)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestIsRetryableError tests checking if errors are retryable
func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "retryable DistKVError",
			err:      New(ErrCodeTimeout, "timeout").WithRetryable(true),
			expected: true,
		},
		{
			name:     "non-retryable DistKVError",
			err:      New(ErrCodeInvalidKey, "invalid").WithRetryable(false),
			expected: false,
		},
		{
			name:     "standard error",
			err:      fmt.Errorf("standard error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryableError(tt.err)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestCommonErrorConstructors tests convenience error constructors
func TestCommonErrorConstructors(t *testing.T) {
	t.Run("NewStorageClosedError", func(t *testing.T) {
		err := NewStorageClosedError()
		if err.Code != ErrCodeStorageClosed {
			t.Error("Expected STORAGE_CLOSED code")
		}
	})

	t.Run("NewKeyNotFoundError", func(t *testing.T) {
		err := NewKeyNotFoundError("user:123")
		if err.Code != ErrCodeKeyNotFound {
			t.Error("Expected KEY_NOT_FOUND code")
		}
		if err.Context["key"] != "user:123" {
			t.Error("Expected key context to be set")
		}
	})

	t.Run("NewQuorumFailedError", func(t *testing.T) {
		err := NewQuorumFailedError(3, 2)
		if err.Code != ErrCodeQuorumFailed {
			t.Error("Expected QUORUM_FAILED code")
		}
		if err.Context["needed"] != 3 {
			t.Error("Expected needed context to be 3")
		}
		if err.Context["got"] != 2 {
			t.Error("Expected got context to be 2")
		}
		if !err.IsRetryable() {
			t.Error("Expected quorum error to be retryable")
		}
	})

	t.Run("NewConnectionError", func(t *testing.T) {
		cause := fmt.Errorf("connection refused")
		err := NewConnectionError("localhost:8080", cause)
		if err.Code != ErrCodeConnectionFailed {
			t.Error("Expected CONNECTION_FAILED code")
		}
		if err.Context["address"] != "localhost:8080" {
			t.Error("Expected address context to be set")
		}
		if !err.IsRetryable() {
			t.Error("Expected connection error to be retryable")
		}
	})

	t.Run("NewTimeoutError", func(t *testing.T) {
		err := NewTimeoutError("database query")
		if err.Code != ErrCodeTimeout {
			t.Error("Expected TIMEOUT code")
		}
		if err.Context["operation"] != "database query" {
			t.Error("Expected operation context to be set")
		}
		if !err.IsRetryable() {
			t.Error("Expected timeout error to be retryable")
		}
	})

	t.Run("NewInvalidConfigError", func(t *testing.T) {
		err := NewInvalidConfigError("invalid port number")
		if err.Code != ErrCodeInvalidConfig {
			t.Error("Expected INVALID_CONFIG code")
		}
		if !strings.Contains(err.Message, "invalid port number") {
			t.Error("Expected custom message to be set")
		}
	})
}

// TestStackTrace tests that stack traces are captured
func TestStackTrace(t *testing.T) {
	err := New(ErrCodeInternal, "test error")

	if err.StackTrace == "" {
		t.Error("Expected stack trace to be captured")
	}

	// Stack trace should contain file and line information
	if !strings.Contains(err.StackTrace, "errors_test.go") {
		t.Error("Expected stack trace to contain test file name")
	}
}

// TestErrorChaining tests chaining multiple error wraps
func TestErrorChaining(t *testing.T) {
	baseErr := fmt.Errorf("base error")
	level1 := Wrap(baseErr, ErrCodeInternal, "level 1")
	level2 := Wrap(level1, ErrCodeConnectionFailed, "level 2")
	level3 := Wrap(level2, ErrCodeQuorumFailed, "level 3")

	// Check that we can unwrap through the chain
	if level3.Code != ErrCodeQuorumFailed {
		t.Error("Expected top level code")
	}

	unwrapped1 := level3.Unwrap()
	if dkvErr, ok := unwrapped1.(*DistKVError); !ok || dkvErr.Code != ErrCodeConnectionFailed {
		t.Error("Expected level 2 code after first unwrap")
	}

	unwrapped2 := unwrapped1.(*DistKVError).Unwrap()
	if dkvErr, ok := unwrapped2.(*DistKVError); !ok || dkvErr.Code != ErrCodeInternal {
		t.Error("Expected level 1 code after second unwrap")
	}

	unwrapped3 := unwrapped2.(*DistKVError).Unwrap()
	if unwrapped3 != baseErr {
		t.Error("Expected base error after third unwrap")
	}
}

// TestContextMutability tests that context modifications mutate in place
func TestContextMutability(t *testing.T) {
	err1 := New(ErrCodeInternal, "test")
	err2 := err1.WithContext("key1", "value1")
	err3 := err2.WithContext("key2", "value2")

	// WithContext mutates in place and returns the same pointer
	if err1 != err2 || err2 != err3 {
		t.Error("Expected all error variables to point to the same instance")
	}

	// All should have both context fields
	if len(err1.Context) != 2 {
		t.Errorf("Expected err1 to have 2 context fields, got %d", len(err1.Context))
	}

	if len(err3.Context) != 2 {
		t.Errorf("Expected err3 to have 2 context fields, got %d", len(err3.Context))
	}

	if err3.Context["key1"] != "value1" || err3.Context["key2"] != "value2" {
		t.Error("Expected both context fields to be set")
	}
}

// TestAllErrorCodes tests that all error codes are defined
func TestAllErrorCodes(t *testing.T) {
	codes := []ErrorCode{
		ErrCodeStorageClosed,
		ErrCodeKeyNotFound,
		ErrCodeInvalidKey,
		ErrCodeInvalidValue,
		ErrCodeStorageCorrupted,
		ErrCodeStorageFull,
		ErrCodeCompactionFailed,
		ErrCodeConnectionFailed,
		ErrCodeTimeout,
		ErrCodeNodeUnreachable,
		ErrCodeQuorumFailed,
		ErrCodeInsufficientNodes,
		ErrCodeReplicationFailed,
		ErrCodeConflict,
		ErrCodeVersionMismatch,
		ErrCodeInvalidConfig,
		ErrCodeInternal,
		ErrCodeUnknown,
	}

	for _, code := range codes {
		err := New(code, "test")
		if err.Code != code {
			t.Errorf("Error code mismatch for %s", code)
		}
	}
}
