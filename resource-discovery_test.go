package ocfsdk

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/ugorji/go/codec"
)

func TestRetrieveResourceDiscovery(t *testing.T) {
	out := `[{"anchor":"ocf://96a718fc-3a65-4751-6602-8ff7b6e5fb40","href":"/oic/d","if":["oic.if.baseline"],"p":{"bm":3},"rt":["oic.wk.d"]},{"anchor":"ocf://96a718fc-3a65-4751-6602-8ff7b6e5fb40/oic/res","href":"/oic/res","if":["oic.if.baseline","oic.if.ll"],"p":{"bm":3},"rel":"self","rt":["oic.wk.res"]}]`
	params := createResourceDeviceParams(t)
	r, err := NewResourceDevice(params)
	if err != nil {
		t.Fatal("cannot create resource device", err)
	}
	d, err := NewResourceDiscovery()
	if err != nil {
		t.Fatal("cannot create resource discovery", err)
	}
	device, err := NewDevice(r, d)
	if err != nil {
		t.Fatal("cannot create device", err)
	}

	payload, err := d.GetResourceOperations().(ResourceOperationRetrieveI).Retrieve(&testRequest{res: d, device: device})
	if err != nil {
		t.Fatal("unexpected error", err)
	}
	if _, ok := payload.([]interface{}); !ok {
		t.Fatalf("unexpected type returns %v", payload)
	}
	bw := new(bytes.Buffer)
	h := new(codec.JsonHandle)
	h.BasicHandle.Canonical = true
	enc := codec.NewEncoder(bw, h)
	err = enc.Encode(payload)
	if err != nil {
		t.Fatal("cannot encode to json", err)
		return
	}

	if out != bw.String() {
		fmt.Printf("'%v' != '%v' !!! \n", out, bw.String())
		t.Fatal("encoded string is not same as pattern")
	}
}
