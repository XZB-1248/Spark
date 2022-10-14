//go:build linux || windows || darwin

package screenshot

import (
	"Spark/client/common"
	"Spark/client/config"
	"bytes"
	"errors"
	"github.com/kbinani/screenshot"
	"image/jpeg"
)

func GetScreenshot(bridge string) error {
	writer := new(bytes.Buffer)
	num := screenshot.NumActiveDisplays()
	if num == 0 {
		err := errors.New(`${i18n|DESKTOP.NO_DISPLAY_FOUND}`)
		return err
	}
	img, err := screenshot.CaptureDisplay(0)
	if err != nil {
		return err
	}
	err = jpeg.Encode(writer, img, &jpeg.Options{Quality: 80})
	if err != nil {
		return err
	}
	url := config.GetBaseURL(false) + `/api/bridge/push`
	_, err = common.HTTP.R().SetBody(writer.Bytes()).SetQueryParam(`bridge`, bridge).Put(url)
	return err
}
