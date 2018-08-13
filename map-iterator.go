package ocfsdk

import (
	"reflect"
	"sort"
)

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

type MapSort struct {
	keys []reflect.Value
}

func (m *MapSort) Len() int {
	return len(m.keys)
}

func (m *MapSort) Less(i, j int) bool {
	//TODO for more types
	return m.keys[i].Interface().(string) < m.keys[j].Interface().(string)
}

func (m *MapSort) Swap(i, j int) {
	tmp := m.keys[i]
	m.keys[i] = m.keys[j]
	m.keys[j] = tmp
}

func NewMapIterator(data map[interface{}]interface{}) MapIteratorI {
	k := &MapSort{keys: reflect.ValueOf(data).MapKeys()}
	sort.Sort(k)
	return &MapIterator{data: data, keys: k.keys, currentIdx: 0, err: nil}
}

type MapIteratorMiddleware struct {
	i MapIteratorI
}

func (m *MapIteratorMiddleware) Next() bool {
	return m.i.Next()
}

func (m *MapIteratorMiddleware) value() interface{} {
	return m.i.value()
}

func (m *MapIteratorMiddleware) Error() error {
	return m.Error()
}
