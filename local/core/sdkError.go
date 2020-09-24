package core

import (
	"fmt"

	"google.golang.org/grpc/codes"
)

type SdkError struct {
	errorCode    codes.Code
	wrappedError error
}

func (e *SdkError) Error() string {
	return fmt.Sprintf("Status code %v caused by : %s", e.errorCode, e.wrappedError)
}

func (e *SdkError) Unwrap() error { return e.wrappedError }

func NewCanceled(e error) *SdkError {
	return &SdkError{
		errorCode:    codes.Canceled,
		wrappedError: e,
	}
}

func NewUnknown(e error) *SdkError {
	return &SdkError{
		errorCode:    codes.Unknown,
		wrappedError: e,
	}
}

func NewInvalidArgument(e error) *SdkError {
	return &SdkError{
		errorCode:    codes.InvalidArgument,
		wrappedError: e,
	}
}

func NewDeadlineExceeded(e error) *SdkError {
	return &SdkError{
		errorCode:    codes.DeadlineExceeded,
		wrappedError: e,
	}
}

func NewNotFound(e error) *SdkError {
	return &SdkError{
		errorCode:    codes.NotFound,
		wrappedError: e,
	}
}

func NewAlreadyExists(e error) *SdkError {
	return &SdkError{
		errorCode:    codes.AlreadyExists,
		wrappedError: e,
	}
}

func NewPermissionDenied(e error) *SdkError {
	return &SdkError{
		errorCode:    codes.PermissionDenied,
		wrappedError: e,
	}
}

func NewResourceExhausted(e error) *SdkError {
	return &SdkError{
		errorCode:    codes.ResourceExhausted,
		wrappedError: e,
	}
}

func NewFailedPrecondition(e error) *SdkError {
	return &SdkError{
		errorCode:    codes.FailedPrecondition,
		wrappedError: e,
	}
}

func NewAborted(e error) *SdkError {
	return &SdkError{
		errorCode:    codes.Aborted,
		wrappedError: e,
	}
}

func NewOutOfRange(e error) *SdkError {
	return &SdkError{
		errorCode:    codes.OutOfRange,
		wrappedError: e,
	}
}

func NewUnimplemented(e error) *SdkError {
	return &SdkError{
		errorCode:    codes.Unimplemented,
		wrappedError: e,
	}
}

func NewInternal(e error) *SdkError {
	return &SdkError{
		errorCode:    codes.Internal,
		wrappedError: e,
	}
}

func NewInternalStr(str string, e error) *SdkError {
	return &SdkError{
		errorCode:    codes.Internal,
		wrappedError: fmt.Errorf(str, e),
	}
}

func NewUnavailable(e error) *SdkError {
	return &SdkError{
		errorCode:    codes.Unavailable,
		wrappedError: e,
	}
}

func NewDataLoss(e error) *SdkError {
	return &SdkError{
		errorCode:    codes.DataLoss,
		wrappedError: e,
	}
}

func NewUnauthenticated(e error) *SdkError {
	return &SdkError{
		errorCode:    codes.Unauthenticated,
		wrappedError: e,
	}
}
