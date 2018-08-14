package ocfsdk

import (
	"reflect"
	"sort"
)

type mapIterator struct {
	data       map[interface{}]interface{}
	keys       []reflect.Value
	currentIdx int
	err        error
}

func (i *mapIterator) Next() bool {
	i.currentIdx++
	if i.currentIdx < len(i.keys) {
		return true
	}
	return false
}

func (i *mapIterator) ValueInterface() interface{} {
	if i.currentIdx < len(i.keys) {
		return i.data[i.keys[i.currentIdx].Interface()]
	}
	i.err = ErrInvalidIterator
	return nil
}

func (i *mapIterator) Err() error {
	return i.err
}

type mapSort struct {
	keys []reflect.Value
}

func (m *mapSort) Len() int {
	return len(m.keys)
}

func (m *mapSort) Less(i, j int) bool {
	//TODO for more types
	return m.keys[i].Interface().(string) < m.keys[j].Interface().(string)
}

func (m *mapSort) Swap(i, j int) {
	tmp := m.keys[i]
	m.keys[i] = m.keys[j]
	m.keys[j] = tmp
}

//NewMapIterator creates iterator over map sorted by keys
func NewMapIterator(data map[interface{}]interface{}) MapIteratorI {
	k := &mapSort{keys: reflect.ValueOf(data).MapKeys()}
	sort.Sort(k)
	return &mapIterator{data: data, keys: k.keys, currentIdx: 0, err: nil}
}
