package core

import (
	"fmt"

	"google.golang.org/grpc/codes"
)

type SdkError struct {
	errorCode    codes.Code
	wrappedError error
}

func (e SdkError) Error() string {
	return fmt.Sprintf("Status code %v caused by : %s", e.errorCode, e.wrappedError)
}

func (e SdkError) Unwrap() error { return e.wrappedError }

func (e SdkError) GetCode() codes.Code {
	return e.errorCode
}

func MakeCanceled(e error) *SdkError {
	return &SdkError{
		errorCode:    codes.Canceled,
		wrappedError: e,
	}
}

func MakeUnknown(e error) *SdkError {
	return &SdkError{
		errorCode:    codes.Unknown,
		wrappedError: e,
	}
}

func MakeInvalidArgument(e error) *SdkError {
	return &SdkError{
		errorCode:    codes.InvalidArgument,
		wrappedError: e,
	}
}

func MakeDeadlineExceeded(e error) *SdkError {
	return &SdkError{
		errorCode:    codes.DeadlineExceeded,
		wrappedError: e,
	}
}

func MakeNotFound(e error) *SdkError {
	return &SdkError{
		errorCode:    codes.NotFound,
		wrappedError: e,
	}
}

func MakeAlreadyExists(e error) *SdkError {
	return &SdkError{
		errorCode:    codes.AlreadyExists,
		wrappedError: e,
	}
}

func MakePermissionDenied(e error) *SdkError {
	return &SdkError{
		errorCode:    codes.PermissionDenied,
		wrappedError: e,
	}
}

func MakeResourceExhausted(e error) *SdkError {
	return &SdkError{
		errorCode:    codes.ResourceExhausted,
		wrappedError: e,
	}
}

func MakeFailedPrecondition(e error) *SdkError {
	return &SdkError{
		errorCode:    codes.FailedPrecondition,
		wrappedError: e,
	}
}

func MakeAborted(e error) *SdkError {
	return &SdkError{
		errorCode:    codes.Aborted,
		wrappedError: e,
	}
}

func MakeOutOfRange(e error) *SdkError {
	return &SdkError{
		errorCode:    codes.OutOfRange,
		wrappedError: e,
	}
}

func MakeUnimplemented(e error) *SdkError {
	return &SdkError{
		errorCode:    codes.Unimplemented,
		wrappedError: e,
	}
}

func MakeInternal(e error) *SdkError {
	return &SdkError{
		errorCode:    codes.Internal,
		wrappedError: e,
	}
}

func MakeInternalStr(str string, e error) *SdkError {
	return &SdkError{
		errorCode:    codes.Internal,
		wrappedError: fmt.Errorf(str, e),
	}
}

func MakeUnavailable(e error) *SdkError {
	return &SdkError{
		errorCode:    codes.Unavailable,
		wrappedError: e,
	}
}

func MakeDataLoss(e error) *SdkError {
	return &SdkError{
		errorCode:    codes.DataLoss,
		wrappedError: e,
	}
}

func MakeUnauthenticated(e error) *SdkError {
	return &SdkError{
		errorCode:    codes.Unauthenticated,
		wrappedError: e,
	}
}
