package ocfsdk

import (
	"testing"
)

func testNewBoolValue(t *testing.T, get func(transaction OCFTransactionI) (bool, error), set func(transaction OCFTransactionI, s bool) error) OCFValueI {
	v, err := NewBoolValue(get, set)
	if err != nil {
		t.Fatal("cannot create new value", err)
	}
	return v
}

func testNewMapValue(t *testing.T, get func(transaction OCFTransactionI) (map[string]interface{}, error), set func(transaction OCFTransactionI, s map[string]interface{}) error) OCFValueI {
	v, err := NewMapValue(get, set)
	if err != nil {
		t.Fatal("cannot create new value", err)
	}
	return v
}

func TestNonCreateBoolValue(t *testing.T) {
	_, err := NewBoolValue(nil, nil)
	if err == nil {
		t.Fatal("expected error", err)
	}
}

func TestBoolValueGet(t *testing.T) {
	b := false
	ob, err := NewBoolValue(func(OCFTransactionI) (bool, error) { return b, nil }, nil)
	if err != nil {
		t.Fatal("cannot create value", err)
	}
	if g, ok := ob.(OCFBoolValueGetI); ok {
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
	ob, err := NewBoolValue(nil, func(t OCFTransactionI, s bool) error { b = s; return nil })
	if err != nil {
		t.Fatal("cannot create value", err)
	}
	if g, ok := ob.(OCFBoolValueSetI); ok {
		if err := g.Set(nil, true); err != nil {
			t.Fatal("failed to set value", err)
		} else if b != true {
			t.Fatal("value is not same", err)
		}
	}
}

func TestNonCreateMapValue(t *testing.T) {
	_, err := NewMapValue(nil, nil)
	if err == nil {
		t.Fatal("expected error", err)
	}
}

func TestMapValueGet(t *testing.T) {
	v := map[string]interface{}{
		"test": true,
	}
	m := testNewMapValue(t, func(OCFTransactionI) (map[string]interface{}, error) { return v, nil }, nil)
	if g, ok := m.(OCFMapValueGetI); ok {
		s1, err := g.Get(nil)
		if err != nil {
			t.Fatal("failed to get value", err)
		}
		used := false
		for key, val := range s1 {
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
	m := testNewMapValue(t, nil, func(t OCFTransactionI, s map[string]interface{}) error { v = s; return nil })
	if g, ok := m.(OCFMapValueSetI); ok {
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
