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

func MakeSdkError(code codes.Code, err error) error {
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

func MakeCanceled(e error) error {
	return MakeSdkError(codes.Canceled, e)

}

func MakeUnknown(e error) error {
	return MakeSdkError(codes.Unknown, e)
}

func MakeInvalidArgument(e error) error {
	return MakeSdkError(codes.InvalidArgument, e)
}

func MakeDeadlineExceeded(e error) error {
	return MakeSdkError(codes.DeadlineExceeded, e)
}

func MakeNotFound(e error) error {
	return MakeSdkError(codes.NotFound, e)
}

func MakeAlreadyExists(e error) error {
	return MakeSdkError(codes.AlreadyExists, e)
}

func MakePermissionDenied(e error) error {
	return MakeSdkError(codes.PermissionDenied, e)

}

func MakeResourceExhausted(e error) error {
	return MakeSdkError(codes.ResourceExhausted, e)
}

func MakeFailedPrecondition(e error) error {
	return MakeSdkError(codes.FailedPrecondition, e)
}

func MakeAborted(e error) error {
	return MakeSdkError(codes.Aborted, e)
}

func MakeOutOfRange(e error) error {
	return MakeSdkError(codes.OutOfRange, e)
}

func MakeUnimplemented(e error) error {
	return MakeSdkError(codes.Unimplemented, e)
}

func MakeInternal(e error) error {
	return MakeSdkError(codes.Internal, e)
}

func MakeInternalStr(str string, e error) error {
	return MakeSdkError(codes.Internal, fmt.Errorf(str, e))
}

func MakeUnavailable(e error) error {
	return MakeSdkError(codes.Unavailable, e)
}

func MakeDataLoss(e error) error {
	return MakeSdkError(codes.DataLoss, e)
}

func MakeUnauthenticated(e error) error {
	return MakeSdkError(codes.Unauthenticated, e)
}
