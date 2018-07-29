package ocfsdk

type OCFValueI interface {
	//Set by type
	//Get by type
}

type OCFValueGetI interface {
	GetValue(transaction OCFTransactionI) (interface{}, error)
}

type OCFValueSetI interface {
	SetDefault(transaction OCFTransactionI) error
	SetValue(transaction OCFTransactionI, s interface{}) error
}

type OCFBoolValueGetI interface {
	OCFValueGetI
	Get(transaction OCFTransactionI) (bool, error)
}

type OCFBoolValueSetI interface {
	OCFValueSetI
	Set(transaction OCFTransactionI, s bool) error
}

type OCFEnumValueGetI interface {
	OCFValueGetI
	Get(transaction OCFTransactionI) (string, error)
}

type OCFEnumValueSetI interface {
	OCFValueSetI
	Set(transaction OCFTransactionI, s string) error
}

type OCFIntValueGetI interface {
	OCFValueGetI
	Get(transaction OCFTransactionI) (int, error)
}

type OCFIntValueSetI interface {
	OCFValueSetI
	Set(transaction OCFTransactionI, s int) error
}

type OCFDoubleValueGetI interface {
	OCFValueGetI
	Get(transaction OCFTransactionI) (float64, error)
}

type OCFDoubleValueSetI interface {
	OCFValueSetI
	Set(transaction OCFTransactionI, s float64) error
}

type OCFStringValueGetI interface {
	OCFValueGetI
	Get(transaction OCFTransactionI) (string, error)
}

type OCFStringValueSetI interface {
	OCFValueSetI
	Set(transaction OCFTransactionI, s string) error
}

type OCFBinaryValueGetI interface {
	OCFValueGetI
	Get(transaction OCFTransactionI) ([]byte, error)
}

type OCFBinaryValueSetI interface {
	OCFValueSetI
	Set(transaction OCFTransactionI, s []byte) error
}

// 1D array
type OCFBoolArrayValueGetI interface {
	OCFValueGetI
	Get(transaction OCFTransactionI) ([]bool, error)
}

type OCFBoolArrayValueSetI interface {
	OCFValueSetI
	Set(transaction OCFTransactionI, s []bool) error
}

type OCFEnumArrayValueGetI interface {
	OCFValueGetI
	Get(transaction OCFTransactionI) ([]string, error)
}

type OCFEnumArrayValueSetI interface {
	OCFValueSetI
	Set(transaction OCFTransactionI, s []string) error
}

type OCFIntArrayValueGetI interface {
	OCFValueGetI
	Get(transaction OCFTransactionI) (int, error)
}

type OCFIntArrayValueSetI interface {
	OCFValueSetI
	Set(transaction OCFTransactionI, s []int) error
}

type OCFDoubleArrayValueGetI interface {
	OCFValueGetI
	Get(transaction OCFTransactionI) ([]float64, error)
}

type OCFDoubleArrayValueSetI interface {
	OCFValueSetI
	Set(transaction OCFTransactionI, s []float64) error
}

type OCFStringArrayValueGetI interface {
	OCFValueGetI
	Get(transaction OCFTransactionI) ([]string, error)
}

type OCFStringArrayValueSetI interface {
	OCFValueSetI
	Set(transaction OCFTransactionI, s []string) error
}

type OCFBinaryArrayValueGetI interface {
	OCFValueGetI
	Get(transaction OCFTransactionI) ([][]byte, error)
}

type OCFBinaryArrayValueSetI interface {
	OCFValueSetI
	Set(transaction OCFTransactionI, s [][]byte) error
}

// 2D array
// TODO

// 3D array
// TODO

type OCFMapValueIteratorI interface {
	Next() bool
	Value() OCFValueI
	Key() string
	Error() error
}

type OCFMapValueI interface {
	NewMapValueIterator() OCFMapValueIteratorI
}

type OCFMapValueGetI interface {
	OCFMapValueI
	OCFValueGetI
	Get(transaction OCFTransactionI) (map[string]interface{}, error)
}

type OCFMapValueSetI interface {
	OCFMapValueI
	OCFValueSetI
	Set(transaction OCFTransactionI, s map[string]interface{}) error
}
