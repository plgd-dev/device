package ocfsdk

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/ugorji/go/codec"
)

func testNewResource(t *testing.T, params *ResourceParams) ResourceI {
	r, err := NewResource(params)
	if err != nil {
		t.Fatal("cannot create new resource", err)
	}
	return r
}

func testNewResourceType(t *testing.T, id string, attributes []AttributeI) ResourceTypeI {
	r, err := NewResourceType(id, attributes)
	if err != nil {
		t.Fatal("cannot create new resource type", err)
	}
	return r
}

func testNewAttribute(t *testing.T, id string, value ValueI, limit LimitI) AttributeI {
	a, err := NewAttribute(id, value, limit)
	if err != nil {
		t.Fatal("cannot create new attribute", err)
	}
	return a
}

type testRequest struct {
	iface   string
	res     ResourceI
	payload interface{}
	device  DeviceI
}

func (t *testRequest) GetResource() ResourceI {
	return t.res
}

func (t *testRequest) GetPayload() PayloadI {
	return t.payload
}

func (t *testRequest) GetInterfaceId() string {
	return t.iface
}

func (t *testRequest) GetQueryParameters() []string {
	return nil
}

func (t *testRequest) GetPeerSession() interface{} {
	return nil
}

func (t *testRequest) GetDevice() DeviceI {
	return t.device
}

func TestCreateResource(t *testing.T) {
	params := ResourceParams{
		Id: "",
	}
	_, err := NewResource(&params)
	if err == nil {
		t.Fatal("expected error")
	}
	params.Id = "test"
	_, err = NewResource(&params)
	if err == nil {
		t.Fatal("expected error")
	}
	params.ResourceInterfaces = []ResourceInterfaceI{}
	params.ResourceTypes = []ResourceTypeI{testNewResourceType(t, "x.test",
		[]AttributeI{
			testNewAttribute(t, "alwaysTrue", testNewValue(t, func(TransactionI) (interface{}, error) { return true, nil }, nil), &BoolLimit{}),
			testNewAttribute(t, "alwaysFalse", testNewValue(t, func(TransactionI) (interface{}, error) { return false, nil }, nil), &BoolLimit{}),
		})}

	_, err = NewResource(&params)
	if err == nil {
		t.Fatal("expected error", err)
	}
	params.ResourceOperations = NewResourceOperationRetrieve(nil)
	_, err = NewResource(&params)
	if err != nil {
		t.Fatal("unexpected error", err)
	}
}

func TestRetrieveResource(t *testing.T) {
	out := `{"alwaysFalse":false,"alwaysTrue":true,"if":["oic.if.baseline"],"rt":["x.test"]}`
	params := ResourceParams{
		Id: "test",
		ResourceTypes: []ResourceTypeI{
			testNewResourceType(t, "x.test",
				[]AttributeI{
					testNewAttribute(t, "alwaysTrue", testNewValue(t, func(TransactionI) (interface{}, error) { return true, nil }, nil), &BoolLimit{}),
					testNewAttribute(t, "alwaysFalse", testNewValue(t, func(TransactionI) (interface{}, error) { return false, nil }, nil), &BoolLimit{}),
				})},
		ResourceOperations: NewResourceOperationRetrieve(func() (TransactionI, error) { return &DummyTransaction{}, nil }),
	}

	r := testNewResource(t, &params)
	payload, err := r.GetResourceOperations().(ResourceOperationRetrieveI).Retrieve(&testRequest{res: r})
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

func TestUpdateResource(t *testing.T) {
	type dataType struct {
		A bool
		B bool
	}
	data := dataType{A: true, B: false}
	params := ResourceParams{
		Id: "test",
		ResourceTypes: []ResourceTypeI{
			testNewResourceType(t, "x.test",
				[]AttributeI{
					testNewAttribute(t, "A", testNewValue(t, nil, func(t TransactionI, s interface{}) error { data.A = s.(bool); return nil }), &BoolLimit{}),
					testNewAttribute(t, "B", testNewValue(t, nil, func(t TransactionI, s interface{}) error { data.B = s.(bool); return nil }), &BoolLimit{}),
				})},
		ResourceOperations: NewResourceOperationUpdate(func() (TransactionI, error) { return &DummyTransaction{}, nil }),
	}

	r := testNewResource(t, &params)
	_, err := r.GetResourceOperations().(ResourceOperationUpdateI).Update(&testRequest{res: r, payload: map[string]interface{}{"A": false, "B": true}})
	if err != nil {
		t.Fatal("unexpected error", err)
	}
	if data.A != false || data.B != true {
		t.Fatal("unexpected values set")
	}
}
