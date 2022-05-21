//go:build linux || windows || darwin

package screenshot

import (
	"Spark/client/config"
	"bytes"
	"errors"
	"github.com/imroc/req/v3"
	"github.com/kbinani/screenshot"
	"image/png"
)

func GetScreenshot(bridge string) error {
	writer := new(bytes.Buffer)
	num := screenshot.NumActiveDisplays()
	if num == 0 {
		err := errors.New(`${i18n|noDisplayFound}`)
		return err
	}
	img, err := screenshot.CaptureDisplay(0)
	if err != nil {
		return err
	}
	err = png.Encode(writer, img)
	if err != nil {
		return err
	}
	url := config.GetBaseURL(false) + `/api/bridge/push`
	_, err = req.R().SetBody(writer.Bytes()).SetQueryParam(`bridge`, bridge).Put(url)
	return err
}
