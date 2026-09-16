package lib

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"syscall"

	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
)

var (
	ErrItemsNotOK                 = errors.New("iaas response contained no data")
	ErrDiskSizeTooLarge           = fmt.Errorf("disk size must be smaller than %v MB", MaxDiskSizeMB)
	ErrNoBlobstoreInAgentSettings = errors.New("no blobstore found in agent settings")
	ErrNoMbusURLInAgentSettings   = errors.New("no messagebus URL found in agent settings")
)

type cpiError struct {
	Err  error
	Ctx  context.Context
	File string
	Line int
}

func (e cpiError) Error() string {
	prefix := "CPI error"
	if e.Ctx != nil {
		method := e.Ctx.Value(cpiMethodKey)
		prefix = fmt.Sprintf("handler=%v", method)
	}

	src := e.File
	if src != "" && e.Line > 0 {
		src = fmt.Sprintf("%s:%d", src, e.Line)
	}
	if src != "" {
		return fmt.Sprintf("%s, %s: %v", src, prefix, e.Err)
	}
	return fmt.Sprintf("%s: %v", prefix, e.Err)
}

func (e cpiError) Unwrap() error { return e.Err }

// Wraps the given error into a customErr struct that includes contextual information
// (request context / source location). The wrapped error is then
// converted into an RPCError used for CPI responses. The Error() method on
// customErr constructs the final message string using the details within the struct.

func WrapErrorInResponse(e error, ctx context.Context) (*RPCResponse, error) {
	// Check if the error IsRetryableOpenAPIError() or the chain contains context.DeadlineExceeded.
	// errors.Is() unwraps all wrapped errors and returns true
	// if any of them are equal to context.DeadlineExceeded.
	isRetryable := IsRetryableOpenAPIError(e) ||
		ContainsTimeoutError(e) ||
		IsNonGenericBuildAbortedNetworkError(e)

	_, file, line, ok := runtime.Caller(1)
	if ok {
		file = filepath.Base(file)
	}
	wrapped := cpiError{
		Err:  e,
		Ctx:  ctx,
		File: file,
		Line: line,
	}

	msg := wrapped.Error()

	rpcError := &RPCError{
		Type:      CloudErrorType,
		Message:   msg,
		OkToRetry: isRetryable,
	}

	switch {
	case errors.Is(wrapped.Err, MethodNotImplementedErrorMessage):
		rpcError.Type = NotImplementedErrorType
	case errors.Is(wrapped.Err, MethodNotSupportedErrorMessage):
		rpcError.Type = NotSupportedErrorType
	}

	return &RPCResponse{
		Result: false,
		Error:  rpcError,
		Log:    msg,
	}, wrapped
}

func IsNonGenericBuildAbortedNetworkError(e error) bool {
	if e == nil {
		return false
	}

	match, err := regexp.MatchString(`Build of instance [a-zA-Z0-9]*-[a-zA-Z0-9]*-[a-zA-Z0-9]*-[a-zA-Z0-9]*-[a-zA-Z0-9]* aborted: Failed to allocate the network\(s\), not rescheduling.`, e.Error())
	if err != nil {
		return false
	}
	return match
}

// Checks if the provided error is of type GenericOpenAPIError and if it is, it will check
// the contained StatusCode against a list of retryable StatusCodes
func IsRetryableOpenAPIError(e error) bool {
	retryableErrorCodes := []int{
		http.StatusRequestTimeout,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout,
	}

	return slices.Contains(retryableErrorCodes, unwrapAndCheckForStatusCode(e)) || IsConnectionResetErr(e)
}

// checks if we see an error when talking to the api. connection resets happen quite often with the IDP
// and unnecessarily break full deployments
func IsConnectionResetErr(e error) bool {
	return errors.Is(e, syscall.ECONNRESET)
}

func ContainsTimeoutError(e error) bool {
	if e == nil {
		return false
	}
	return errors.Is(e, context.DeadlineExceeded) || ContainsTimeoutError(errors.Unwrap(e))
}

func ContainsOpenAPIErrorWithStatusCode(e error, statusCode int) bool {
	return unwrapAndCheckForStatusCode(e) == statusCode
}

func unwrapAndCheckForStatusCode(e error) int {
	if e == nil {
		return 0
	}
	testErr := e

	if err, ok := testErr.(oapierror.GenericOpenAPIError); ok {
		return err.StatusCode
	}
	if err, ok := testErr.(*oapierror.GenericOpenAPIError); ok {
		return err.StatusCode
	}
	return unwrapAndCheckForStatusCode(errors.Unwrap(testErr))
}
