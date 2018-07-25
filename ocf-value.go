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

type OCFArrayValueGetI interface {
	Get() ([]interface{}, error)
}

type OCFArrayValueSetI interface {
	Set(s []interface{}) (changed bool, err error)
}

type OCFMapValueGetI interface {
	Get() (map[string]interface{}, error)
}

type OCFMapValueSetI interface {
	Set(s map[string]interface{}) (changed bool, err error)
}
