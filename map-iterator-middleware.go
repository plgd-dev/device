package ocfsdk

//MapIteratorMiddleware defines middleware for map iterator
type MapIteratorMiddleware struct {
	i MapIteratorI
}

//Next increment iterator and return true for success
func (m *MapIteratorMiddleware) Next() bool {
	return m.i.Next()
}

//ValueInterface returns value of type interface{}
func (m *MapIteratorMiddleware) ValueInterface() interface{} {
	return m.i.ValueInterface()
}

//Err get error of iterator
func (m *MapIteratorMiddleware) Err() error {
	return m.Err()
}
