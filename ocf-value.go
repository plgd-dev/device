package main

type OCFValueI interface {
}

type OCFBoolValueGetI interface {
	Get() (bool, error)
}

type OCFBoolValueSetI interface {
	Set(s bool) (changed bool, err error)
}

type OCFEnumValueGetI interface {
	Get() (string, error)
}

type OCFEnumValueSetI interface {
	Set(s string) (changed bool, err error)
}

type OCFIntValueGetI interface {
	Get() (int, error)
}

type OCFIntValueSetI interface {
	Set(s int) (changed bool, err error)
}

type OCFDoubleValueGetI interface {
	Get() (float64, error)
}

type OCFDoubleValueSetI interface {
	Set(s float64) (changed bool, err error)
}

type OCFStringValueGetI interface {
	Get() (string, error)
}

type OCFStringValueSetI interface {
	Set(s string) (changed bool, err error)
}

type OCFBinaryValueGetI interface {
	Get() ([]byte, error)
}

type OCFBinaryValueSetI interface {
	Set(s []byte) (changed bool, err error)
}

// 1D array
type OCFBoolArrayValueGetI interface {
	Get() ([]bool, error)
}

type OCFBoolArrayValueSetI interface {
	Set(s []bool) (changed bool, err error)
}

type OCFEnumArrayValueGetI interface {
	Get() ([]string, error)
}

type OCFEnumArrayValueSetI interface {
	Set(s []string) (changed bool, err error)
}

type OCFIntArrayValueGetI interface {
	Get() (int, error)
}

type OCFIntArrayValueSetI interface {
	Set(s []int) (changed bool, err error)
}

type OCFDoubleArrayValueGetI interface {
	Get() ([]float64, error)
}

type OCFDoubleArrayValueSetI interface {
	Set(s []float64) (changed bool, err error)
}

type OCFStringArrayValueGetI interface {
	Get() ([]string, error)
}

type OCFStringArrayValueSetI interface {
	Set(s []string) (changed bool, err error)
}

type OCFBinaryArrayValueGetI interface {
	Get() ([][]byte, error)
}

type OCFBinaryArrayValueSetI interface {
	Set(s [][]byte) (changed bool, err error)
}

// 2D array
// TODO

// 3D array
// TODO

type OCFMapValueI {
	OCFValueI
	Unset() error
}

type OCFMapValueGetI interface {
	Get() (map[string]OCFMapValueI, error)
}