//+build linux windows

package screenshot

import (
	"bytes"
	"errors"
	"github.com/kbinani/screenshot"
	"image/png"
)

func GetScreenshot(trigger string) error {
	writer := new(bytes.Buffer)
	num := screenshot.NumActiveDisplays()
	if num == 0 {
		err := errors.New(`no display found`)
		putScreenshot(trigger, err.Error(), nil)
		return err
	}
	img, err := screenshot.CaptureDisplay(0)
	if err != nil {
		putScreenshot(trigger, err.Error(), nil)
		return err
	}
	err = png.Encode(writer, img)
	if err != nil {
		putScreenshot(trigger, err.Error(), nil)
		return err
	}
	_, err = putScreenshot(trigger, ``, writer)
	return err
}
