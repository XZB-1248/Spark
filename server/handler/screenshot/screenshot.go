package screenshot

import (
	"Spark/modules"
	"Spark/server/common"
	"Spark/server/handler/bridge"
	"Spark/server/handler/utility"
	"Spark/utils"
	"Spark/utils/melody"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

// GetScreenshot will call client to screenshot.
func GetScreenshot(ctx *gin.Context) {
	target, ok := utility.CheckForm(ctx, nil)
	if !ok {
		return
	}
	bridgeID := utils.GetStrUUID()
	trigger := utils.GetStrUUID()
	wait := make(chan bool)
	called := false
	common.SendPackByUUID(modules.Packet{Code: 0, Act: `screenshot`, Data: gin.H{`bridge`: bridgeID}, Event: trigger}, target)
	common.AddEvent(func(p modules.Packet, _ *melody.Session) {
		called = true
		bridge.RemoveBridge(bridgeID)
		common.RemoveEvent(trigger)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, modules.Packet{Code: 1, Msg: p.Msg})
		common.Warn(ctx, `TAKE_SCREENSHOT`, `fail`, p.Msg, nil)
		wait <- false
	}, target, trigger)
	instance := bridge.AddBridgeWithDst(nil, bridgeID, ctx)
	instance.OnPush = func(bridge *bridge.Bridge) {
		called = true
		common.RemoveEvent(trigger)
		ctx.Header(`Content-Type`, `image/png`)
	}
	instance.OnFinish = func(bridge *bridge.Bridge) {
		if called {
			common.Info(ctx, `TAKE_SCREENSHOT`, `success`, ``, nil)
		}
		wait <- false
	}
	select {
	case <-wait:
	case <-time.After(5 * time.Second):
		if !called {
			bridge.RemoveBridge(bridgeID)
			common.RemoveEvent(trigger)
			ctx.AbortWithStatusJSON(http.StatusGatewayTimeout, modules.Packet{Code: 1, Msg: `${i18n|COMMON.RESPONSE_TIMEOUT}`})
			common.Warn(ctx, `TAKE_SCREENSHOT`, `fail`, `timeout`, nil)
		} else {
			<-wait
		}
	}
	close(wait)
}
