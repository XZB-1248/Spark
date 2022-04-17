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
	// Prevent constant overflow when GOARCH is arm or i386.
	_, _, err := syscall.Syscall(syscall.SYS_REBOOT, syscall.LINUX_REBOOT_CMD_HALT, 0, 0)
	return err
}

func Suspend() error {
	// Prevent constant overflow when GOARCH is arm or i386.
	_, _, err := syscall.Syscall(syscall.SYS_REBOOT, syscall.LINUX_REBOOT_CMD_SW_SUSPEND, 0, 0)
	return err
}

func Restart() error {
	return syscall.Reboot(syscall.LINUX_REBOOT_CMD_RESTART)
}

func Shutdown() error {
	return syscall.Reboot(syscall.LINUX_REBOOT_CMD_POWER_OFF)
}
