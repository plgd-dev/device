package ocfsdk

type MapIteratorI interface {
	Next() bool
	Error() error

	value() interface{}
}
