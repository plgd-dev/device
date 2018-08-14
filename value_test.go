package ocfsdk

import (
	"testing"
)

func testNewValue(t *testing.T, get func(transaction TransactionI) (PayloadI, error), set func(transaction TransactionI, s PayloadI) error) ValueI {
	v, err := NewValue(get, set)
	if err != nil {
		t.Fatal("cannot create new value", err)
	}
	return v
}

func TestNonCreateValue(t *testing.T) {
	_, err := NewValue(nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBoolValueGet(t *testing.T) {
	b := false
	ob, err := NewValue(func(TransactionI) (PayloadI, error) { return b, nil }, nil)
	if err != nil {
		t.Fatal("cannot create value", err)
	}
	if g, ok := ob.(ValueGetI); ok {
		if v, err := g.Get(nil); err != nil {
			t.Fatal("failed to get value", err)
		} else if v != b {
			t.Fatal("value is not same", err)
		}
	} else {
		t.Fatal("not implement interface", err)
	}
}

func TestBoolValueSet(t *testing.T) {
	b := false
	ob, err := NewValue(nil, func(t TransactionI, s PayloadI) error { b = s.(bool); return nil })
	if err != nil {
		t.Fatal("cannot create value", err)
	}
	if g, ok := ob.(ValueSetI); ok {
		if err := g.Set(nil, true); err != nil {
			t.Fatal("failed to set value", err)
		} else if b != true {
			t.Fatal("value is not same", err)
		}
	}
}

func TestMapValueGet(t *testing.T) {
	v := map[string]interface{}{
		"test": true,
	}
	m := testNewValue(t, func(TransactionI) (PayloadI, error) { return v, nil }, nil)
	if g, ok := m.(ValueGetI); ok {
		s1, err := g.Get(nil)
		if err != nil {
			t.Fatal("failed to get value", err)
		}
		used := false
		for key, val := range s1.(map[string]interface{}) {
			used = true
			if key != "test" {
				t.Fatal("invalid key", err)
			}
			if g, ok := val.(bool); ok {
				if g != true {
					t.Fatal("value is not same", err)
				}
			} else {
				t.Fatal("not implement interface", err)
			}
		}
		if !used {
			t.Fatal("empty map", err)
		}
	} else {
		t.Fatal("not implement interface")
	}
}

func TestMapValueSet(t *testing.T) {
	v := map[string]interface{}{
		"test": false,
	}
	m := testNewValue(t, nil, func(t TransactionI, s PayloadI) error { v = s.(map[string]interface{}); return nil })
	if g, ok := m.(ValueSetI); ok {
		err := g.Set(nil, map[string]interface{}{
			"test1": 123,
		})
		if err != nil {
			t.Fatal("failed to get value", err)
		}
		used := false
		for key, val := range v {
			used = true
			if key != "test1" {
				t.Fatal("invalid key", err)
			}
			if g, ok := val.(int); ok {
				if g != 123 {
					t.Fatal("value is not same", err)
				}
			} else {
				t.Fatal("not implement interface", err)
			}
		}
		if !used {
			t.Fatal("empty map", err)
		}
	} else {
		t.Fatal("not implement interface")
	}
}
