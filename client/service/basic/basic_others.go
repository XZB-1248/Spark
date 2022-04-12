//go:build !linux && !windows
// +build !linux,!windows

package basic

import "errors"

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
	return errors.New(`${i18n|operationNotSupported}`)
}

func Shutdown() error {
	return errors.New(`${i18n|operationNotSupported}`)
}
