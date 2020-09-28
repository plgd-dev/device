package core

import (
	"errors"
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

func MakeSdkError(code codes.Code, err error) SdkError {
	var orig SdkError
	if errors.As(err, &orig) {
		return err
	} else {
		return SdkError{
			errorCode:    code,
			wrappedError: err,
		}
	}
}

func MakeCanceled(e error) SdkError {
	return MakeSdkError(codes.Canceled, e)

}

func MakeUnknown(e error) SdkError {
	return MakeSdkError(codes.Unknown, e)
}

func MakeInvalidArgument(e error) SdkError {
	return MakeSdkError(codes.InvalidArgument, e)
}

func MakeDeadlineExceeded(e error) SdkError {
	return MakeSdkError(codes.DeadlineExceeded, e)
}

func MakeNotFound(e error) SdkError {
	return MakeSdkError(codes.NotFound, e)
}

func MakeAlreadyExists(e error) SdkError {
	return MakeSdkError(codes.AlreadyExists, e)
}

func MakePermissionDenied(e error) SdkError {
	return MakeSdkError(codes.PermissionDenied, e)

}

func MakeResourceExhausted(e error) SdkError {
	return MakeSdkError(codes.ResourceExhausted, e)
}

func MakeFailedPrecondition(e error) SdkError {
	return MakeSdkError(codes.FailedPrecondition, e)
}

func MakeAborted(e error) SdkError {
	return MakeSdkError(codes.Aborted, e)
}

func MakeOutOfRange(e error) SdkError {
	return MakeSdkError(codes.OutOfRange, e)
}

func MakeUnimplemented(e error) SdkError {
	return MakeSdkError(codes.Unimplemented, e)
}

func MakeInternal(e error) SdkError {
	return MakeSdkError(codes.Internal, e)
}

func MakeInternalStr(str string, e error) SdkError {
	return MakeSdkError(codes.Internal, fmt.Errorf(str, e))
}

func MakeUnavailable(e error) SdkError {
	return MakeSdkError(codes.Unavailable, e)
}

func MakeDataLoss(e error) SdkError {
	return MakeSdkError(codes.DataLoss, e)
}

func MakeUnauthenticated(e error) SdkError {
	return MakeSdkError(codes.Unauthenticated, e)
}
