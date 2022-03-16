// +build linux

package basic

import (
	"errors"
	"syscall"
)

func init() {
}

func Lock() error {
	return errors.New(`the operation is not supported`)
}

func Logoff() error {
	return errors.New(`the operation is not supported`)
}

func Hibernate() error {
	_, _, err := syscall.Syscall(syscall.SYS_REBOOT, syscall.LINUX_REBOOT_CMD_HALT, 0, 0)
	return err
}

func Suspend() error {
	_, _, err := syscall.Syscall(syscall.SYS_REBOOT, syscall.LINUX_REBOOT_CMD_SW_SUSPEND, 0, 0)
	return err
}

func Restart() error {
	return syscall.Reboot(syscall.LINUX_REBOOT_CMD_RESTART)
}

func Shutdown() error {
	return syscall.Reboot(syscall.LINUX_REBOOT_CMD_POWER_OFF)
}
