//go:build !linux && !windows && !darwin

package basic

import (
	"errors"
	"os/exec"
)

func init() {
}

func Lock() error {
	return errors.New(`${i18n|COMMON.OPERATION_NOT_SUPPORTED}`)
}

func Logoff() error {
	return errors.New(`${i18n|COMMON.OPERATION_NOT_SUPPORTED}`)
}

func Hibernate() error {
	return errors.New(`${i18n|COMMON.OPERATION_NOT_SUPPORTED}`)
}

func Suspend() error {
	return errors.New(`${i18n|COMMON.OPERATION_NOT_SUPPORTED}`)
}

func Restart() error {
	return exec.Command(`reboot`).Run()
}

func Shutdown() error {
	return exec.Command(`shutdown`).Run()
}
