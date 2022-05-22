//go:build !linux && !windows && !darwin

package screenshot

import "errors"

func GetScreenshot(bridge string) error {
	return errors.New(`${i18n|operationNotSupported}`)
}
