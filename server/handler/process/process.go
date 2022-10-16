package process

import (
	"Spark/modules"
	"Spark/server/common"
	"Spark/server/handler/utility"
	"Spark/utils"
	"Spark/utils/melody"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

// ListDeviceProcesses will list processes on remote client
func ListDeviceProcesses(ctx *gin.Context) {
	connUUID, ok := utility.CheckForm(ctx, nil)
	if !ok {
		return
	}
	trigger := utils.GetStrUUID()
	common.SendPackByUUID(modules.Packet{Act: `PROCESSES_LIST`, Event: trigger}, connUUID)
	ok = common.AddEventOnce(func(p modules.Packet, _ *melody.Session) {
		if p.Code != 0 {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, modules.Packet{Code: 1, Msg: p.Msg})
		} else {
			ctx.JSON(http.StatusOK, modules.Packet{Code: 0, Data: p.Data})
		}
	}, connUUID, trigger, 5*time.Second)
	if !ok {
		ctx.AbortWithStatusJSON(http.StatusGatewayTimeout, modules.Packet{Code: 1, Msg: `${i18n|COMMON.RESPONSE_TIMEOUT}`})
	}
}

// KillDeviceProcess will try to get send a packet to
// client and let it kill the process specified.
func KillDeviceProcess(ctx *gin.Context) {
	var form struct {
		Pid int32 `json:"pid" yaml:"pid" form:"pid" binding:"required"`
	}
	target, ok := utility.CheckForm(ctx, &form)
	if !ok {
		return
	}
	trigger := utils.GetStrUUID()
	common.SendPackByUUID(modules.Packet{Act: `PROCESS_KILL`, Data: gin.H{`pid`: form.Pid}, Event: trigger}, target)
	ok = common.AddEventOnce(func(p modules.Packet, _ *melody.Session) {
		if p.Code != 0 {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, modules.Packet{Code: 1, Msg: p.Msg})
			common.Warn(ctx, `PROCESS_KILL`, `fail`, p.Msg, map[string]any{
				`pid`: form.Pid,
			})
		} else {
			ctx.JSON(http.StatusOK, modules.Packet{Code: 0})
			common.Info(ctx, `PROCESS_KILL`, `success`, ``, map[string]any{
				`pid`: form.Pid,
			})
		}
	}, target, trigger, 5*time.Second)
	if !ok {
		ctx.AbortWithStatusJSON(http.StatusGatewayTimeout, modules.Packet{Code: 1, Msg: `${i18n|COMMON.RESPONSE_TIMEOUT}`})
		common.Warn(ctx, `PROCESS_KILL`, `fail`, `timeout`, map[string]any{
			`pid`: form.Pid,
		})
	}
}
