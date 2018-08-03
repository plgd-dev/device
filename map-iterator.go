package ocfsdk

import "reflect"

type MapIterator struct {
	data       map[interface{}]interface{}
	keys       []reflect.Value
	currentIdx int
	err        error
}

func (i *MapIterator) Next() bool {
	i.currentIdx++
	if i.currentIdx < len(i.keys) {
		return true
	}
	return false
}

func (i *MapIterator) value() interface{} {
	if i.currentIdx < len(i.keys) {
		return i.data[i.keys[i.currentIdx].Interface()]
	}
	i.err = ErrInvalidIterator
	return nil
}

func (i *MapIterator) Error() error {
	return i.err
}
