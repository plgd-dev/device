package ocfsdk

type ValueI interface {
	//Set by type
	//Get by type
}

type ValueGetI interface {
	GetValue(transaction TransactionI) (interface{}, error)
}

type ValueSetI interface {
	SetValue(transaction TransactionI, s interface{}) error
}

type BoolValueGetI interface {
	ValueGetI
	Get(transaction TransactionI) (bool, error)
}

type BoolValueSetI interface {
	ValueSetI
	Set(transaction TransactionI, s bool) error
}

type EnumValueGetI interface {
	ValueGetI
	Get(transaction TransactionI) (string, error)
}

type EnumValueSetI interface {
	ValueSetI
	Set(transaction TransactionI, s string) error
}

type IntValueGetI interface {
	ValueGetI
	Get(transaction TransactionI) (int, error)
}

type IntValueSetI interface {
	ValueSetI
	Set(transaction TransactionI, s int) error
}

type DoubleValueGetI interface {
	ValueGetI
	Get(transaction TransactionI) (float64, error)
}

type DoubleValueSetI interface {
	ValueSetI
	Set(transaction TransactionI, s float64) error
}

type StringValueGetI interface {
	ValueGetI
	Get(transaction TransactionI) (string, error)
}

type StringValueSetI interface {
	ValueSetI
	Set(transaction TransactionI, s string) error
}

type BinaryValueGetI interface {
	ValueGetI
	Get(transaction TransactionI) ([]byte, error)
}

type BinaryValueSetI interface {
	ValueSetI
	Set(transaction TransactionI, s []byte) error
}

// 1D array
type BoolArrayValueGetI interface {
	ValueGetI
	Get(transaction TransactionI) ([]bool, error)
}

type BoolArrayValueSetI interface {
	ValueSetI
	Set(transaction TransactionI, s []bool) error
}

type EnumArrayValueGetI interface {
	ValueGetI
	Get(transaction TransactionI) ([]string, error)
}

type EnumArrayValueSetI interface {
	ValueSetI
	Set(transaction TransactionI, s []string) error
}

type IntArrayValueGetI interface {
	ValueGetI
	Get(transaction TransactionI) (int, error)
}

type IntArrayValueSetI interface {
	ValueSetI
	Set(transaction TransactionI, s []int) error
}

type DoubleArrayValueGetI interface {
	ValueGetI
	Get(transaction TransactionI) ([]float64, error)
}

type DoubleArrayValueSetI interface {
	ValueSetI
	Set(transaction TransactionI, s []float64) error
}

type StringArrayValueGetI interface {
	ValueGetI
	Get(transaction TransactionI) ([]string, error)
}

type StringArrayValueSetI interface {
	ValueSetI
	Set(transaction TransactionI, s []string) error
}

type BinaryArrayValueGetI interface {
	ValueGetI
	Get(transaction TransactionI) ([][]byte, error)
}

type BinaryArrayValueSetI interface {
	ValueSetI
	Set(transaction TransactionI, s [][]byte) error
}

type MapArrayValueGetI interface {
	ValueGetI
	Get(transaction TransactionI) ([]map[string]interface{}, error)
}

type MapArrayValueSetI interface {
	ValueSetI
	Set(transaction TransactionI, value []map[string]interface{}) error
}

// 2D array
// TODO

// 3D array
// TODO

type MapValueGetI interface {
	ValueGetI
	Get(transaction TransactionI) (map[string]interface{}, error)
}

type MapValueSetI interface {
	ValueSetI
	Set(transaction TransactionI, value map[string]interface{}) error
}
