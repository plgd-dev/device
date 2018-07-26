package main

type OCFValueI interface {
	//Set by type
	//Get by type
}

type OCFValueSetDefaultI interface {
	SetDefault() error
}

type OCFBoolValueGetI interface {
	Get() (bool, error)
}

type OCFBoolValueSetI interface {
	OCFValueSetDefaultI
	Set(s bool) (changed bool, err error)
}

type OCFEnumValueGetI interface {
	Get() (string, error)
}

type OCFEnumValueSetI interface {
	OCFValueSetDefaultI
	Set(s string) (changed bool, err error)
}

type OCFIntValueGetI interface {
	Get() (int, error)
}

type OCFIntValueSetI interface {
	OCFValueSetDefaultI
	Set(s int) (changed bool, err error)
}

type OCFDoubleValueGetI interface {
	Get() (float64, error)
}

type OCFDoubleValueSetI interface {
	OCFValueSetDefaultI
	Set(s float64) (changed bool, err error)
}

type OCFStringValueGetI interface {
	Get() (string, error)
}

type OCFStringValueSetI interface {
	OCFValueSetDefaultI
	Set(s string) (changed bool, err error)
}

type OCFBinaryValueGetI interface {
	Get() ([]byte, error)
}

type OCFBinaryValueSetI interface {
	OCFValueSetDefaultI
	Set(s []byte) (changed bool, err error)
}

// 1D array
type OCFBoolArrayValueGetI interface {
	Get() ([]bool, error)
}

type OCFBoolArrayValueSetI interface {
	OCFValueSetDefaultI
	Set(s []bool) (changed bool, err error)
}

type OCFEnumArrayValueGetI interface {
	Get() ([]string, error)
}

type OCFEnumArrayValueSetI interface {
	OCFValueSetDefaultI
	Set(s []string) (changed bool, err error)
}

type OCFIntArrayValueGetI interface {
	Get() (int, error)
}

type OCFIntArrayValueSetI interface {
	OCFValueSetDefaultI
	Set(s []int) (changed bool, err error)
}

type OCFDoubleArrayValueGetI interface {
	Get() ([]float64, error)
}

type OCFDoubleArrayValueSetI interface {
	OCFValueSetDefaultI
	Set(s []float64) (changed bool, err error)
}

type OCFStringArrayValueGetI interface {
	Get() ([]string, error)
}

type OCFStringArrayValueSetI interface {
	OCFValueSetDefaultI
	Set(s []string) (changed bool, err error)
}

type OCFBinaryArrayValueGetI interface {
	Get() ([][]byte, error)
}

type OCFBinaryArrayValueSetI interface {
	OCFValueSetDefaultI
	Set(s [][]byte) (changed bool, err error)
}

// 2D array
// TODO

// 3D array
// TODO

type OCFMapValueGetI interface {
	Get() (map[string]OCFValueI, error)
}
