// +build windows

package basic

import (
	"syscall"
	"unsafe"
)

func init() {
	privilege()
}

func privilege() error {
	user32 := syscall.MustLoadDLL("user32")
	defer user32.Release()
	kernel32 := syscall.MustLoadDLL("kernel32")
	defer user32.Release()
	advapi32 := syscall.MustLoadDLL("advapi32")
	defer advapi32.Release()

	GetLastError := kernel32.MustFindProc("GetLastError")
	GetCurrentProcess := kernel32.MustFindProc("GetCurrentProcess")
	OpenProdcessToken := advapi32.MustFindProc("OpenProcessToken")
	LookupPrivilegeValue := advapi32.MustFindProc("LookupPrivilegeValueW")
	AdjustTokenPrivileges := advapi32.MustFindProc("AdjustTokenPrivileges")

	currentProcess, _, _ := GetCurrentProcess.Call()

	const tokenAdjustPrivileges = 0x0020
	const tokenQuery = 0x0008
	var hToken uintptr

	result, _, err := OpenProdcessToken.Call(currentProcess, tokenAdjustPrivileges|tokenQuery, uintptr(unsafe.Pointer(&hToken)))
	if result != 1 {
		return err
	}

	const SeShutdownName = "SeShutdownPrivilege"

	type Luid struct {
		lowPart  uint32 // DWORD
		highPart int32  // long
	}
	type LuidAndAttributes struct {
		luid       Luid   // LUID
		attributes uint32 // DWORD
	}

	type TokenPrivileges struct {
		privilegeCount uint32 // DWORD
		privileges     [1]LuidAndAttributes
	}

	var tkp TokenPrivileges

	utf16ptr, err := syscall.UTF16PtrFromString(SeShutdownName)
	if err != nil {
		return err
	}

	result, _, err = LookupPrivilegeValue.Call(uintptr(0), uintptr(unsafe.Pointer(utf16ptr)), uintptr(unsafe.Pointer(&(tkp.privileges[0].luid))))
	if result != 1 {
		return err
	}

	const SePrivilegeEnabled uint32 = 0x00000002

	tkp.privilegeCount = 1
	tkp.privileges[0].attributes = SePrivilegeEnabled

	result, _, err = AdjustTokenPrivileges.Call(hToken, 0, uintptr(unsafe.Pointer(&tkp)), 0, uintptr(0), 0)
	if result != 1 {
		return err
	}

	result, _, _ = GetLastError.Call()
	if result != 0 {
		return err
	}

	return nil
}

func Lock() error {
	dll := syscall.MustLoadDLL(`user32`)
	_, _, err := dll.MustFindProc(`LockWorkStation`).Call()
	dll.Release()
	if err == syscall.Errno(0) {
		return nil
	}
	return err
}

func Logoff() error {
	dll := syscall.MustLoadDLL(`user32`)
	_, _, err := dll.MustFindProc(`ExitWindowsEx`).Call(0x0, 0x0)
	dll.Release()
	if err == syscall.Errno(0) {
		return nil
	}
	return err
}

func Hibernate() error {
	dll := syscall.MustLoadDLL(`powrprof`)
	_, _, err := dll.MustFindProc(`SetSuspendState`).Call(0x0, 0x0, 0x1)
	dll.Release()
	if err == syscall.Errno(0) {
		return nil
	}
	return err
}

func Suspend() error {
	dll := syscall.MustLoadDLL(`powrprof`)
	_, _, err := dll.MustFindProc(`SetSuspendState`).Call(0x1, 0x0, 0x1)
	dll.Release()
	if err == syscall.Errno(0) {
		return nil
	}
	return err
}

func Restart() error {
	dll := syscall.MustLoadDLL(`user32`)
	_, _, err := dll.MustFindProc(`ExitWindowsEx`).Call(0x2, 0x0)
	dll.Release()
	if err == syscall.Errno(0) {
		return nil
	}
	return err
}

func Shutdown() error {
	dll := syscall.MustLoadDLL(`user32`)
	_, _, err := dll.MustFindProc(`ExitWindowsEx`).Call(0x1, 0x0)
	dll.Release()
	if err == syscall.Errno(0) {
		return nil
	}
	return err
}
