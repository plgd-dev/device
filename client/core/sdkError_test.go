// ************************************************************************
// Copyright (C) 2024 plgd.dev, s.r.o.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// ************************************************************************

package core_test

import (
	"errors"
	"testing"

	"github.com/plgd-dev/device/v2/client/core"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

var err = errors.New("test")

func TestMakeCanceled(t *testing.T) {
	sdkErr := core.SdkError{}
	require.True(t, errors.As(core.MakeCanceled(err), &sdkErr))
	require.Equal(t, codes.Canceled, sdkErr.GetCode())
	require.Equal(t, err, sdkErr.Unwrap())
}

func TestMakeUnknown(t *testing.T) {
	sdkErr := core.SdkError{}
	require.True(t, errors.As(core.MakeUnknown(err), &sdkErr))
	require.Equal(t, codes.Unknown, sdkErr.GetCode())
	require.Equal(t, err, sdkErr.Unwrap())
}

func TestMakeInvalidArgument(t *testing.T) {
	sdkErr := core.SdkError{}
	require.True(t, errors.As(core.MakeInvalidArgument(err), &sdkErr))
	require.Equal(t, codes.InvalidArgument, sdkErr.GetCode())
	require.Equal(t, err, sdkErr.Unwrap())
}

func TestMakeDeadlineExceeded(t *testing.T) {
	sdkErr := core.SdkError{}
	require.True(t, errors.As(core.MakeDeadlineExceeded(err), &sdkErr))
	require.Equal(t, codes.DeadlineExceeded, sdkErr.GetCode())
	require.Equal(t, err, sdkErr.Unwrap())
}

func TestMakeNotFound(t *testing.T) {
	sdkErr := core.SdkError{}
	require.True(t, errors.As(core.MakeNotFound(err), &sdkErr))
	require.Equal(t, codes.NotFound, sdkErr.GetCode())
	require.Equal(t, err, sdkErr.Unwrap())
}

func TestMakeAlreadyExists(t *testing.T) {
	sdkErr := core.SdkError{}
	require.True(t, errors.As(core.MakeAlreadyExists(err), &sdkErr))
	require.Equal(t, codes.AlreadyExists, sdkErr.GetCode())
	require.Equal(t, err, sdkErr.Unwrap())
}

func TestMakePermissionDenied(t *testing.T) {
	sdkErr := core.SdkError{}
	require.True(t, errors.As(core.MakePermissionDenied(err), &sdkErr))
	require.Equal(t, codes.PermissionDenied, sdkErr.GetCode())
	require.Equal(t, err, sdkErr.Unwrap())
}

func TestMakeResourceExhausted(t *testing.T) {
	sdkErr := core.SdkError{}
	require.True(t, errors.As(core.MakeResourceExhausted(err), &sdkErr))
	require.Equal(t, codes.ResourceExhausted, sdkErr.GetCode())
	require.Equal(t, err, sdkErr.Unwrap())
}

func TestMakeFailedPrecondition(t *testing.T) {
	sdkErr := core.SdkError{}
	require.True(t, errors.As(core.MakeFailedPrecondition(err), &sdkErr))
	require.Equal(t, codes.FailedPrecondition, sdkErr.GetCode())
	require.Equal(t, err, sdkErr.Unwrap())
}

func TestMakeAborted(t *testing.T) {
	sdkErr := core.SdkError{}
	require.True(t, errors.As(core.MakeAborted(err), &sdkErr))
	require.Equal(t, codes.Aborted, sdkErr.GetCode())
	require.Equal(t, err, sdkErr.Unwrap())
}

func TestMakeOutOfRange(t *testing.T) {
	sdkErr := core.SdkError{}
	require.True(t, errors.As(core.MakeOutOfRange(err), &sdkErr))
	require.Equal(t, codes.OutOfRange, sdkErr.GetCode())
	require.Equal(t, err, sdkErr.Unwrap())
}

func TestMakeUnimplemented(t *testing.T) {
	sdkErr := core.SdkError{}
	require.True(t, errors.As(core.MakeUnimplemented(err), &sdkErr))
	require.Equal(t, codes.Unimplemented, sdkErr.GetCode())
	require.Equal(t, err, sdkErr.Unwrap())
}

func TestMakeInternal(t *testing.T) {
	sdkErr := core.SdkError{}
	require.True(t, errors.As(core.MakeInternal(err), &sdkErr))
	require.Equal(t, codes.Internal, sdkErr.GetCode())
	require.Equal(t, err, sdkErr.Unwrap())
}

func TestMakeInternalStr(t *testing.T) {
	sdkErr := core.SdkError{}
	require.True(t, errors.As(core.MakeInternalStr("test:%v", err), &sdkErr))
	require.Equal(t, codes.Internal, sdkErr.GetCode())
	require.Contains(t, sdkErr.Error(), "test:"+err.Error())
}

func TestMakeUnavailable(t *testing.T) {
	sdkErr := core.SdkError{}
	require.True(t, errors.As(core.MakeUnavailable(err), &sdkErr))
	require.Equal(t, codes.Unavailable, sdkErr.GetCode())
	require.Equal(t, err, sdkErr.Unwrap())
}

func TestMakeDataLoss(t *testing.T) {
	sdkErr := core.SdkError{}
	require.True(t, errors.As(core.MakeDataLoss(err), &sdkErr))
	require.Equal(t, codes.DataLoss, sdkErr.GetCode())
	require.Equal(t, err, sdkErr.Unwrap())
}

func TestMakeUnauthenticated(t *testing.T) {
	sdkErr := core.SdkError{}
	require.True(t, errors.As(core.MakeUnauthenticated(err), &sdkErr))
	require.Equal(t, codes.Unauthenticated, sdkErr.GetCode())
	require.Equal(t, err, sdkErr.Unwrap())
}
