package ocfsdk

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/ugorji/go/codec"
)

func testNewResource(t *testing.T, id string, discoverable bool, observeable bool, resourceTypes []ResourceTypeI, resourceInterfaces []ResourceInterfaceI, openTransaction func() (TransactionI, error)) ResourceI {
	r, err := NewResource(id, discoverable, observeable, resourceTypes, resourceInterfaces, openTransaction)
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

func TestCreateResource(t *testing.T) {
	_, err := NewResource("", false, false, nil, nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	_, err = NewResource("test", false, false, nil, nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}

	_, err = NewResource("test", false, false, []ResourceTypeI{
		testNewResourceType(t, "x.test",
			[]AttributeI{
				testNewAttribute(t, "alwaysTrue", testNewBoolValue(t, func(TransactionI) (bool, error) { return true, nil }, nil), &BoolLimit{}),
				testNewAttribute(t, "alwaysFalse", testNewBoolValue(t, func(TransactionI) (bool, error) { return false, nil }, nil), &BoolLimit{}),
			})}, nil, nil)
	if err != nil {
		t.Fatal("unexpected error", err)
	}
}

func TestRetrieveResource(t *testing.T) {
	out := `{"alwaysFalse":false,"alwaysTrue":true,"if":["oic.if.baseline"],"rt":["x.test"]}`

	r := testNewResource(t, "test", false, false, []ResourceTypeI{
		testNewResourceType(t, "x.test",
			[]AttributeI{
				testNewAttribute(t, "alwaysTrue", testNewBoolValue(t, func(TransactionI) (bool, error) { return true, nil }, nil), &BoolLimit{}),
				testNewAttribute(t, "alwaysFalse", testNewBoolValue(t, func(TransactionI) (bool, error) { return false, nil }, nil), &BoolLimit{}),
			})}, nil, nil)
	payload, _, err := r.(ResourceRetrieveI).Retrieve(&testRequest{iface: "", res: r})
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

	r := testNewResource(t, "test", false, false, []ResourceTypeI{
		testNewResourceType(t, "x.test",
			[]AttributeI{
				testNewAttribute(t, "A", testNewBoolValue(t, nil, func(t TransactionI, s bool) error { data.A = s; return nil }), &BoolLimit{}),
				testNewAttribute(t, "B", testNewBoolValue(t, nil, func(t TransactionI, s bool) error { data.B = s; return nil }), &BoolLimit{}),
			})}, nil, nil)
	_, _, err := r.(ResourceUpdateI).Update(&testRequest{iface: "", res: r, payload: map[string]interface{}{"A": false, "B": true}})
	if err != nil {
		t.Fatal("unexpected error", err)
	}
	if data.A != false || data.B != true {
		t.Fatal("unexpected values set")
	}
}
