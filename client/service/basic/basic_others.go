//go:build !linux && !windows && !darwin

package basic

import (
	"errors"
	"os/exec"
)

func init() {
}

func Lock() error {
	return errors.New(`${i18n|operationNotSupported}`)
}

func Logoff() error {
	return errors.New(`${i18n|operationNotSupported}`)
}

func Hibernate() error {
	return errors.New(`${i18n|operationNotSupported}`)
}

func Suspend() error {
	return errors.New(`${i18n|operationNotSupported}`)
}

func Restart() error {
	return exec.Command(`reboot`).Run()
}

func Shutdown() error {
	return exec.Command(`shutdown`).Run()
}
