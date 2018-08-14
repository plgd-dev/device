package ocfsdk

//MapIteratorI defines iterator over map
type MapIteratorI interface {
	//Next increment iterator and return true for success
	Next() bool
	//Err get error of iterator
	Err() error
	//ValueInterface returns value of type interface{}
	ValueInterface() interface{}
}
