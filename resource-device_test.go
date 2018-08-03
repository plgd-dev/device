package ocfsdk

import (
	"bytes"
	"fmt"
	"testing"

	uuid "github.com/nu7hatch/gouuid"
	"github.com/ugorji/go/codec"
)

func TestNonCreateResourceDevice(t *testing.T) {
	params := &ResourceDeviceParams{}
	if _, err := NewResourceDevice(params); err == nil {
		t.Fatal("expected error")
	}
	di, err := uuid.NewV4()
	if err != nil {
		t.Fatalf("cannot create uuid")
	}
	params.DeviceId = di
	if _, err := NewResourceDevice(params); err == nil {
		t.Fatal("expected error")
	}
	params.ProtocolIndependentID = di
	if _, err := NewResourceDevice(params); err == nil {
		t.Fatal("expected error")
	}
	params.DeviceName = "test"
	if _, err := NewResourceDevice(params); err == nil {
		t.Fatal("expected error")
	}
	params.SpecVersion = "test"
	if _, err := NewResourceDevice(params); err == nil {
		t.Fatal("expected error")
	}
	params.DataModelVersion = "test"
	if _, err := NewResourceDevice(params); err != nil {
		t.Fatal("unexpected error", err)
	}
}

func createResourceDeviceParams(t *testing.T) *ResourceDeviceParams {
	di, err := uuid.ParseHex("96a718fc-3a65-4751-6602-8ff7b6e5fb40")
	if err != nil {
		t.Fatal("cannot create uuid", err)
	}
	piid, err := uuid.ParseHex("0b1d5406-a145-4d88-4447-091bcf1f22ad")
	if err != nil {
		t.Fatal("cannot create uuid", err)
	}
	params := &ResourceDeviceParams{
		DeviceId:              di,
		ProtocolIndependentID: piid,
		DeviceName:            "DeviceName",
		SpecVersion:           "SpecVersion",
		DataModelVersion:      "DataModelVersion",
		ModelNumber:           "ModelNumber",
		ManufacturerName:      []string{"ManufacturerName"},
	}
	return params
}

func TestCreateResourceDevice(t *testing.T) {
	params := createResourceDeviceParams(t)
	dr, err := NewResourceDevice(params)
	if err != nil {
		t.Fatal("cannot create device", err)
	}
	if str, err := dr.GetDeviceName(); str != params.DeviceName || err != nil {
		t.Fatalf("invalid value %v != %v: %v", str, params.DeviceName, err)
	}
	if str, err := dr.GetDataModelVersion(); str != params.DataModelVersion || err != nil {
		t.Fatalf("invalid value %v != %v: %v", str, params.DataModelVersion, err)
	}
	if str, err := dr.GetSpecVersion(); str != params.SpecVersion || err != nil {
		t.Fatalf("invalid value %v != %v: %v", str, params.SpecVersion, err)
	}
	if str, err := dr.GetModelNumber(); str != params.ModelNumber || err != nil {
		t.Fatalf("invalid value %v != %v: %v", str, params.ModelNumber, err)
	}
	if str, err := dr.GetDeviceId(); str.String() != params.DeviceId.String() || err != nil {
		t.Fatalf("invalid value %v != %v: %v", str.String(), params.DeviceId.String(), err)
	}
	if str, err := dr.GetProtocolIndependentID(); str.String() != params.ProtocolIndependentID.String() || err != nil {
		t.Fatalf("invalid value %v != %v: %v", str.String(), params.ProtocolIndependentID.String(), err)
	}
	str, err := dr.GetManufacturerName()
	if err != nil {
		t.Fatal("cannot get value", err)
	}
	if str[0] != params.ManufacturerName[0] {
		t.Fatalf("invalid value %v != %v", str[0], params.ManufacturerName[0])
	}

}

func TestRetrieveResourceDevice(t *testing.T) {
	out := `{"di":"96a718fc-3a65-4751-6602-8ff7b6e5fb40","dmn":["ManufacturerName"],"dmno":"ModelNumber","dmv":"DataModelVersion","icv":"SpecVersion","if":["oic.if.baseline"],"n":"DeviceName","piid":"0b1d5406-a145-4d88-4447-091bcf1f22ad","rt":["oic.wk.d"]}`
	params := createResourceDeviceParams(t)
	r, err := NewResourceDevice(params)
	if err != nil {
		t.Fatal("cannot create resource device", err)
	}
	payload, _, err := r.GetResourceOperations().(ResourceOperationRetrieveI).Retrieve(&testRequest{iface: "", res: r})
	if err != nil {
		t.Fatal("unexpected error", err)
	}
	if _, ok := payload.(map[string]interface{}); !ok {
		t.Fatal("unexpected type returns")
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
