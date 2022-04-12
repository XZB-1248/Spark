//go:build linux
// +build linux

package basic

import (
	"errors"
	"syscall"
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
	return syscall.Reboot(syscall.LINUX_REBOOT_CMD_HALT)
}

func Suspend() error {
	return syscall.Reboot(syscall.LINUX_REBOOT_CMD_SW_SUSPEND)
}

func Restart() error {
	return syscall.Reboot(syscall.LINUX_REBOOT_CMD_RESTART)
}

func Shutdown() error {
	return syscall.Reboot(syscall.LINUX_REBOOT_CMD_POWER_OFF)
}
