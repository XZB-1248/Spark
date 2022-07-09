//go:build darwin
// +build darwin

package basic

/*
#cgo LDFLAGS: -framework CoreServices -framework Carbon
#include <CoreServices/CoreServices.h>
#include <Carbon/Carbon.h>

static OSStatus SendAppleEventToSystemProcess(AEEventID EventToSend);

OSStatus SendAppleEventToSystemProcess(AEEventID EventToSend)
{
    AEAddressDesc targetDesc;
    static const ProcessSerialNumber kPSNOfSystemProcess = { 0, kSystemProcess };
    AppleEvent eventReply = {typeNull, NULL};
    AppleEvent appleEventToSend = {typeNull, NULL};

    OSStatus error = noErr;

    error = AECreateDesc(typeProcessSerialNumber, &kPSNOfSystemProcess, sizeof(kPSNOfSystemProcess), &targetDesc);

    if (error != noErr) {
        return(error);
    }

    error = AECreateAppleEvent(kCoreEventClass, EventToSend, &targetDesc, kAutoGenerateReturnID, kAnyTransactionID, &appleEventToSend);

    AEDisposeDesc(&targetDesc);
    if (error != noErr) {
        return(error);
    }

    error = AESend(&appleEventToSend, &eventReply, kAENoReply, kAENormalPriority, kAEDefaultTimeout, NULL, NULL);

    AEDisposeDesc(&appleEventToSend);
    if (error != noErr) {
        return(error);
    }

    AEDisposeDesc(&eventReply);

    return(error);
}
*/
import "C"

import (
	"errors"
	"os/exec"
)

// I'm not familiar with macOS, that's all I can do.
func init() {
}

func Lock() error {
	return errors.New(`${i18n|operationNotSupported}`)
}

func Logoff() error {
	if C.SendAppleEventToSystemProcess(C.kAEReallyLogOut) == C.noErr {
		return nil
	} else {
		return errors.New(`${i18n|operationNotSupported}`)
	}
}

func Hibernate() error {
	if C.SendAppleEventToSystemProcess(C.kAESleep) == C.noErr {
		return nil
	} else {
		return errors.New(`${i18n|operationNotSupported}`)
	}
}

func Suspend() error {
	return errors.New(`${i18n|operationNotSupported}`)
}

func Restart() error {
	if C.SendAppleEventToSystemProcess(C.kAERestart) == C.noErr {
		return nil
	} else {
		return exec.Command(`reboot`).Run()
	}
}

func Shutdown() error {
	if C.SendAppleEventToSystemProcess(C.kAEShutDown) == C.noErr {
		return nil
	} else {
		return exec.Command(`shutdown`).Run()
	}
}
