package screenshot

import (
	"Spark/client/config"
	"github.com/imroc/req/v3"
)

func putScreenshot(trigger, err string, body interface{}) (*req.Response, error) {
	return req.R().
		SetBody(body).
		SetHeaders(map[string]string{
			`Trigger`: trigger,
			`Error`:   err,
		}).
		Send(`PUT`, config.GetBaseURL(false)+`/api/device/screenshot/put`)
}
