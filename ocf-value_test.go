package ocfsdk

import "testing"

func testNewBoolValue(t *testing.T, get func(transaction OCFTransactionI) (bool, error), setDefault func(transaction OCFTransactionI) error, set func(transaction OCFTransactionI, s bool) error) OCFValueI {
	v, err := NewBoolValue(get, setDefault, set)
	if err != nil {
		t.Fatal("cannot create new value: %v", err)
	}
	return v
}

func TestCreateBoolValue(t *testing.T) {
	_, err := NewBoolValue(nil, nil, nil)
	if err == nil {
		t.Fatal("expected error", err)
	}
}

func TestBoolValueGet(t *testing.T) {
	b := false
	ob, err := NewBoolValue(func(OCFTransactionI) (bool, error) { return b, nil }, nil, nil)
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

func TestBoolValueSetDefault(t *testing.T) {
	b := false
	ob, err := NewBoolValue(nil, func(OCFTransactionI) error { b = true; return nil }, nil)
	if err != nil {
		t.Fatal("cannot create value", err)
	}
	if g, ok := ob.(OCFBoolValueSetI); ok {
		if err := g.SetDefault(nil); err != nil {
			t.Fatal("failed to get value", err)
		} else if b != true {
			t.Fatal("value is not same", err)
		}
	} else {
		t.Fatal("not implement interface", err)
	}
}

func TestBoolValueSet(t *testing.T) {
	b := false
	ob, err := NewBoolValue(nil, nil, func(t OCFTransactionI, s bool) error { b = s; return nil })
	if err != nil {
		t.Fatal("cannot create value", err)
	}
	if g, ok := ob.(OCFBoolValueSetI); ok {
		if err := g.Set(nil, true); err != nil {
			t.Fatal("failed to set value", err)
		}
	}
}

func TestMapValue(t *testing.T) {
	v, err := NewBoolValue(func(OCFTransactionI) (bool, error) { return true, nil }, nil, nil)
	if err != nil {
		t.Fatal("cannot create value", err)
	}
	s := map[string]OCFValueI{
		"test": v,
	}

	m, err := NewMapValue(func(OCFTransactionI) (map[string]OCFValueI, error) { return s, nil })
	if err != nil {
		t.Fatal("cannot create value", err)
	}
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
			if g, ok := val.(OCFBoolValueGetI); ok {
				if v, err := g.Get(nil); err != nil {
					t.Fatal("failed to get value", err)
				} else if v != true {
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
		t.Fatal("not implement interface", err)
	}
}
