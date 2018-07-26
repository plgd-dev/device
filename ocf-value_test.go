package main

import "testing"

func TestCreateBoolValue(t *testing.T) {
	_, err := NewBoolValue(nil, nil, nil)
	if err == nil {
		t.Fatal("expected error", err)
	}
}

func TestBoolValueGet(t *testing.T) {
	b := false
	ob, err := NewBoolValue(func() (bool, error) { return b, nil }, nil, nil)
	if err != nil {
		t.Fatal("cannot create value", err)
	}
	if g, ok := ob.(OCFBoolValueGetI); ok {
		if v, err := g.Get(); err != nil {
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
	ob, err := NewBoolValue(nil, func() error { b = true; return nil }, nil)
	if err != nil {
		t.Fatal("cannot create value", err)
	}
	if g, ok := ob.(OCFBoolValueSetI); ok {
		if err := g.SetDefault(); err != nil {
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
	ob, err := NewBoolValue(nil, nil, func(s bool) (bool, error) { c := s != b; b = s; return c, nil })
	if err != nil {
		t.Fatal("cannot create value", err)
	}
	if g, ok := ob.(OCFBoolValueSetI); ok {
		if changed, err := g.Set(true); err != nil {
			t.Fatal("failed to get value", err)
		} else if !changed {
			t.Fatal("value was not changed", err)
		}
	}
	if g, ok := ob.(OCFBoolValueSetI); ok {
		if changed, err := g.Set(true); err != nil {
			t.Fatal("failed to get value", err)
		} else if changed {
			t.Fatal("value was changed", err)
		}
	} else {
		t.Fatal("not implement interface", err)
	}
}

func TestMapValue(t *testing.T) {
	v, err := NewBoolValue(func() (bool, error) { return true, nil }, nil, nil)
	if err != nil {
		t.Fatal("cannot create value", err)
	}
	s := map[string]OCFValueI{
		"test": v,
	}

	m, err := NewMapValue(s)
	if err != nil {
		t.Fatal("cannot create value", err)
	}
	if g, ok := m.(OCFMapValueGetI); ok {
		s1, err := g.Get()
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
				if v, err := g.Get(); err != nil {
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
