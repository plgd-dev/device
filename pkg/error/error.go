package error

import "fmt"

func NotSupported() error {
	return fmt.Errorf("not supported")
}
