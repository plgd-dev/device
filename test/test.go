package test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/go-ocf/kit/codec/cbor"

	"github.com/go-ocf/sdk/app"
	local "github.com/go-ocf/sdk/localEx"
	"github.com/stretchr/testify/require"
)

type observationOB struct {
	cancel context.CancelFunc
	t      *testing.T
}

type kicOb struct {
	Status int `codec:"status"`
}

func (o *observationOB) Handle(ctx context.Context, body []byte) {
	var v kicOb
	err := cbor.Decode(body, &v)
	require.NoError(o.t, err)
	fmt.Println("obc status: ", v.Status)
	if v.Status == 0 {
		o.cancel()
	}
}

func (o *observationOB) OnClose() {
	o.cancel()
	require.NoError(o.t, fmt.Errorf("connection closed"))
}

func (o *observationOB) Error(err error) {
	require.NoError(o.t, err)
}

func MustGetHostname() string {
	n, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	return n
}

func MustFindDeviceByName(name string) (deviceID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	deviceID, err := FindDeviceByName(ctx, name)
	if err != nil {
		panic(err)
	}
	return deviceID
}

func FindDeviceByName(ctx context.Context, name string) (deviceID string, _ error) {
	appCallback, err := app.NewApp(nil)
	if err != nil {
		return "", fmt.Errorf("cannot create app callback: %w", err)
	}
	client, err := local.NewClientFromConfig(&local.Config{}, appCallback, func(error) {})
	if err != nil {
		return "", fmt.Errorf("could not find the device named %s: %w", name, err)
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	err = client.GetDeviceDetails(ctx, []string{"oic.wk.d"}, func(error) {}, func(d local.DeviceDetails) {
		if d.Device.Name == name {
			deviceID = d.ID
			cancel()
		}
	})
	if err != nil {
		return "", fmt.Errorf("could not find the device named %s: %w", name, err)
	}
	if deviceID == "" {
		return "", fmt.Errorf("could not find the device named %s: not found", name)
	}
	return deviceID, nil
}

func DecodeCbor(t *testing.T, data []byte) interface{} {
	var v interface{}
	err := cbor.Decode(data, &v)
	require.NoError(t, err)
	return v
}

func EncodeToCbor(t *testing.T, v interface{}) []byte {
	d, err := cbor.Encode(v)
	require.NoError(t, err)
	return d
}
