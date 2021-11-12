package client_test

import (
	"context"
	"testing"
	"time"

	"github.com/plgd-dev/device/client"
	"github.com/plgd-dev/device/schema/device"
	"github.com/plgd-dev/device/test"
	"github.com/stretchr/testify/require"
)

/*
func TestClientDiscoveryBatch(t *testing.T) {
	deviceID := test.MustFindDeviceByName("aa")
	c, err := NewTestSecureClient()
	require.NoError(t, err)
	defer func() {
		err := c.Close(context.Background())
		require.NoError(t, err)
	}()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3600)
	defer cancel()
	deviceID, err = c.OwnDevice(ctx, deviceID, client.WithOTM(client.OTMType_JustWorks))
	require.NoError(t, err)
	defer c.DisownDevice(ctx, deviceID)

	var wg sync.WaitGroup

	o1 := makeObservationHandler()
	id1, err := c.ObserveResource(ctx, deviceID, "/oic/res", o1, client.WithInterface("oic.if.baseline"))
	require.NoError(t, err)
	stopBaseline := func() {
		fmt.Printf("stopping %v /oic/res?if=oic.if.baseline", id1)
		//c.StopObservingResource(ctx, id1)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for f := range o1.res {
			var v interface{}
			err := f(&v)
			require.NoError(t, err)
			val, err := json.Encode(v)
			require.NoError(t, err)
			fmt.Printf("baseline discovery observation: %v\n--------------\n", string(val))
		}
	}()

	o2 := makeObservationHandler()
	id2, err := c.ObserveResource(ctx, deviceID, "/oic/res", o2)
	require.NoError(t, err)
	defer func() {
		fmt.Printf("stopping %v /oic/res", id2)
		// c.StopObservingResource(ctx, id2)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for f := range o2.res {
			var v interface{}
			err := f(&v)
			require.NoError(t, err)
			val, err := json.Encode(v)
			require.NoError(t, err)
			fmt.Printf("default discovery observation: %v\n--------------\n", string(val))
		}
	}()

	o := makeObservationHandler()
	id, err := c.ObserveResource(ctx, deviceID, "/oic/res", o, client.WithInterface("oic.if.b"))
	require.NoError(t, err)
	defer func() {
		fmt.Printf("stopping %v /oic/res?if=oic.if.b", id)
		// c.StopObservingResource(ctx, id)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for f := range o.res {
			var v interface{}
			err := f(&v)
			require.NoError(t, err)
			val, err := json.Encode(v)
			require.NoError(t, err)
			fmt.Printf("batch observation: %v\n--------------\n", string(val))
		}
	}()

	for i := uint64(0); i < 1; i++ {
		fmt.Printf("%v\n", i)
		v := make([]byte, 0, 1)
		for i := 0; i < cap(v); i++ {
			v = append(v, 'a')
		}

		err = c.UpdateResource(ctx, deviceID, "/oc/con", map[string]interface{}{
			"n": fmt.Sprintf("devname-%v-%v", i, string(v)),
		}, nil)
		require.NoError(t, err)
	}

	o3 := makeObservationHandler()
	id3, err := c.ObserveResource(ctx, deviceID, "/oc/con", o3)
	require.NoError(t, err)
	stop := func() {
		fmt.Printf("stopping %v /oc/con", id3)
		c.StopObservingResource(ctx, id3)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for f := range o2.res {
			var v interface{}
			err := f(&v)
			require.NoError(t, err)
			val, err := json.Encode(v)
			require.NoError(t, err)
			fmt.Printf("default observation /oc/con: %v\n--------------\n", string(val))
		}
	}()

	err = c.UpdateResource(ctx, deviceID, "/oc/con", map[string]interface{}{
		"n": "aa",
	}, nil)
	require.NoError(t, err)
	stop()

	time.Sleep(time.Second)

	//create
	err = c.CreateResource(ctx, deviceID, "/switches", map[string]interface{}{
		"rt": []string{"oic.r.switch.binary"},
		"if": []string{"oic.if.a", "oic.if.baseline"},
		"rep": map[string]interface{}{
			"value": false,
		},
		"p": map[string]interface{}{
			"bm": uint64(3),
		},
	}, nil)
	require.NoError(t, err)

	//delete
	err = c.DeleteResource(ctx, deviceID, "/switches/1", nil)
	require.NoError(t, err)

	//create
	err = c.CreateResource(ctx, deviceID, "/switches", map[string]interface{}{
		"rt": []string{"oic.r.switch.binary"},
		"if": []string{"oic.if.a", "oic.if.baseline"},
		"rep": map[string]interface{}{
			"value": false,
		},
		"p": map[string]interface{}{
			"bm": uint64(3),
		},
	}, nil)
	require.NoError(t, err)
	//create
	err = c.CreateResource(ctx, deviceID, "/switches", map[string]interface{}{
		"rt": []string{"oic.r.switch.binary"},
		"if": []string{"oic.if.a", "oic.if.baseline"},
		"rep": map[string]interface{}{
			"value": false,
		},
		"p": map[string]interface{}{
			"bm": uint64(3),
		},
	}, nil)
	require.NoError(t, err)

	time.Sleep(time.Second)

	stopBaseline()
}
*/

func TestClientOwnDevice(t *testing.T) {
	_ = test.MustFindDeviceByName(test.DevsimNetHost)
	type args struct {
		deviceName string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				deviceName: test.DevsimNetHost,
			},
		},
	}

	c, err := NewTestSecureClient()
	require.NoError(t, err)
	defer func() {
		err := c.Close(context.Background())
		require.NoError(t, err)
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
			defer cancel()
			deviceID, err := test.FindDeviceByName(ctx, tt.args.deviceName)
			require.NoError(t, err)
			deviceID, err = c.OwnDevice(ctx, deviceID)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			device1, err := c.GetDeviceByMulticast(ctx, deviceID)
			require.NoError(t, err)
			require.Equal(t, device1.OwnershipStatus, client.OwnershipStatus_Owned)
			err = c.DisownDevice(ctx, deviceID)
			require.NoError(t, err)
			_ = test.MustFindDeviceByName(tt.args.deviceName)
			deviceID, err = test.FindDeviceByName(ctx, tt.args.deviceName)
			require.NoError(t, err)
			deviceID, err = c.OwnDevice(ctx, deviceID)
			require.NoError(t, err)
			device2, err := c.GetDeviceByMulticast(ctx, deviceID)
			require.NoError(t, err)
			require.Equal(t, device1.Details.(*device.Device).ProtocolIndependentID, device2.Details.(*device.Device).ProtocolIndependentID)
			require.Equal(t, device1.OwnershipStatus, client.OwnershipStatus_Owned)
			err = c.DisownDevice(ctx, deviceID)
			require.NoError(t, err)
		})
	}
}
