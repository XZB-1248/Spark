//+build !linux
//+build !windows

package screenshot

import "Spark/utils"

func GetScreenshot(trigger string) error {
	_, err := putScreenshot(trigger, utils.ErrUnsupported.Error(), nil)
	return err
}