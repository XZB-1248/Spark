// +build !linux
// +build !windows

package basic

import "errors"

func init() {
}

func Lock() error {
	return errors.New(`the operation is not supported`)
}

func Logoff() error {
	return errors.New(`the operation is not supported`)
}

func Hibernate() error {
	return errors.New(`the operation is not supported`)
}

func Suspend() error {
	return errors.New(`the operation is not supported`)
}

func Restart() error {
	return errors.New(`the operation is not supported`)
}

func Shutdown() error {
	return errors.New(`the operation is not supported`)
}
