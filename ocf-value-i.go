package ocfsdk

type OCFValueI interface {
	//Set by type
	//Get by type
}

type OCFValueSetDefaultI interface {
	SetDefault(transaction OCFTransactionI) error
}

type OCFBoolValueGetI interface {
	Get(transaction OCFTransactionI) (bool, error)
}

type OCFBoolValueSetI interface {
	OCFValueSetDefaultI
	Set(transaction OCFTransactionI, s bool) error
}

type OCFEnumValueGetI interface {
	Get(transaction OCFTransactionI) (string, error)
}

type OCFEnumValueSetI interface {
	OCFValueSetDefaultI
	Set(transaction OCFTransactionI, s string) error
}

type OCFIntValueGetI interface {
	Get(transaction OCFTransactionI) (int, error)
}

type OCFIntValueSetI interface {
	OCFValueSetDefaultI
	Set(transaction OCFTransactionI, s int) error
}

type OCFDoubleValueGetI interface {
	Get(transaction OCFTransactionI) (float64, error)
}

type OCFDoubleValueSetI interface {
	OCFValueSetDefaultI
	Set(transaction OCFTransactionI, s float64) error
}

type OCFStringValueGetI interface {
	Get(transaction OCFTransactionI) (string, error)
}

type OCFStringValueSetI interface {
	OCFValueSetDefaultI
	Set(transaction OCFTransactionI, s string) error
}

type OCFBinaryValueGetI interface {
	Get(transaction OCFTransactionI) ([]byte, error)
}

type OCFBinaryValueSetI interface {
	OCFValueSetDefaultI
	Set(transaction OCFTransactionI, s []byte) error
}

// 1D array
type OCFBoolArrayValueGetI interface {
	Get(transaction OCFTransactionI) ([]bool, error)
}

type OCFBoolArrayValueSetI interface {
	OCFValueSetDefaultI
	Set(transaction OCFTransactionI, s []bool) error
}

type OCFEnumArrayValueGetI interface {
	Get(transaction OCFTransactionI) ([]string, error)
}

type OCFEnumArrayValueSetI interface {
	OCFValueSetDefaultI
	Set(transaction OCFTransactionI, s []string) error
}

type OCFIntArrayValueGetI interface {
	Get(transaction OCFTransactionI) (int, error)
}

type OCFIntArrayValueSetI interface {
	OCFValueSetDefaultI
	Set(transaction OCFTransactionI, s []int) error
}

type OCFDoubleArrayValueGetI interface {
	Get(transaction OCFTransactionI) ([]float64, error)
}

type OCFDoubleArrayValueSetI interface {
	OCFValueSetDefaultI
	Set(transaction OCFTransactionI, s []float64) error
}

type OCFStringArrayValueGetI interface {
	Get(transaction OCFTransactionI) ([]string, error)
}

type OCFStringArrayValueSetI interface {
	OCFValueSetDefaultI
	Set(transaction OCFTransactionI, s []string) error
}

type OCFBinaryArrayValueGetI interface {
	Get(transaction OCFTransactionI) ([][]byte, error)
}

type OCFBinaryArrayValueSetI interface {
	OCFValueSetDefaultI
	Set(transaction OCFTransactionI, s [][]byte) error
}

// 2D array
// TODO

// 3D array
// TODO

type OCFMapValueGetI interface {
	Get(transaction OCFTransactionI) (map[string]OCFValueI, error)
}
