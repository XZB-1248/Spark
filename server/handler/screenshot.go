package handler

import (
	"Spark/modules"
	"Spark/server/common"
	"Spark/utils"
	"Spark/utils/melody"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

// putScreenshot will forward screenshot image from client to browser.
func putScreenshot(ctx *gin.Context) {
	errMsg := ctx.GetHeader(`Error`)
	trigger := ctx.GetHeader(`Trigger`)
	if len(trigger) == 0 {
		ctx.JSON(http.StatusBadRequest, modules.Packet{Code: -1, Msg: `${i18n|invalidParameter}`})
		return
	}
	if len(errMsg) > 0 {
		common.CallEvent(modules.Packet{
			Code:  1,
			Msg:   fmt.Sprintf(`${i18n|screenshotFailed}: %v`, errMsg),
			Event: trigger,
		}, nil)
		ctx.JSON(http.StatusOK, modules.Packet{Code: 0})
		return
	}
	data, err := ctx.GetRawData()
	if len(data) == 0 {
		msg := ``
		if err != nil {
			msg = fmt.Sprintf(`${i18n|screenshotObtainFailed}: %v`, err)
			ctx.JSON(http.StatusInternalServerError, modules.Packet{Code: 1, Msg: msg})
		} else {
			msg = `${i18n|screenshotFailed}: ${i18n|unknownError}`
			ctx.JSON(http.StatusOK, modules.Packet{Code: 0})
		}
		common.CallEvent(modules.Packet{
			Code:  1,
			Msg:   msg,
			Event: trigger,
		}, nil)
		return
	}
	common.CallEvent(modules.Packet{
		Code: 0,
		Data: map[string]interface{}{
			`screenshot`: data,
		},
		Event: trigger,
	}, nil)
	ctx.JSON(http.StatusOK, modules.Packet{Code: 0})
}

// getScreenshot will call client to screenshot.
func getScreenshot(ctx *gin.Context) {
	target, ok := checkForm(ctx, nil)
	if !ok {
		return
	}
	trigger := utils.GetStrUUID()
	common.SendPackUUID(modules.Packet{Code: 0, Act: `screenshot`, Event: trigger}, target)
	ok = common.AddEventOnce(func(p modules.Packet, _ *melody.Session) {
		if p.Code != 0 {
			ctx.JSON(http.StatusInternalServerError, modules.Packet{Code: 1, Msg: p.Msg})
		} else {
			data, ok := p.Data[`screenshot`]
			if !ok {
				ctx.JSON(http.StatusInternalServerError, modules.Packet{Code: 1, Msg: `${i18n|screenshotObtainFailed}`})
				return
			}
			screenshot, ok := data.([]byte)
			if !ok {
				ctx.JSON(http.StatusInternalServerError, modules.Packet{Code: 1, Msg: `${i18n|screenshotObtainFailed}`})
				return
			}
			ctx.Data(200, `image/png`, screenshot)
		}
	}, target, trigger, 5*time.Second)
	if !ok {
		ctx.JSON(http.StatusGatewayTimeout, modules.Packet{Code: 1, Msg: `${i18n|responseTimeout}`})
	}
}
