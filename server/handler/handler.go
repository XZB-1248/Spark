package handler

import (
	"Spark/modules"
	"Spark/server/common"
	"github.com/gin-gonic/gin"
	"net/http"
)

var AuthHandler gin.HandlerFunc

// InitRouter will initialize http and websocket routers.
func InitRouter(ctx *gin.RouterGroup) {
	ctx.Any(`/bridge/push`, bridgePush)
	ctx.Any(`/bridge/pull`, bridgePull)
	ctx.Any(`/client/update`, checkUpdate) // Client, for update.
	group := ctx.Group(`/`, AuthHandler)
	{
		group.POST(`/device/screenshot/get`, getScreenshot)
		group.POST(`/device/process/list`, listDeviceProcesses)
		group.POST(`/device/process/kill`, killDeviceProcess)
		group.POST(`/device/file/remove`, removeDeviceFiles)
		group.POST(`/device/file/upload`, uploadToDevice)
		group.POST(`/device/file/list`, listDeviceFiles)
		group.POST(`/device/file/text`, getDeviceTextFile)
		group.POST(`/device/file/get`, getDeviceFiles)
		group.POST(`/device/list`, getDevices)
		group.POST(`/device/:act`, callDevice)
		group.POST(`/client/check`, checkClient)
		group.POST(`/client/generate`, generateClient)
		group.Any(`/device/terminal`, initTerminal) // Browser, handle websocket events for web terminal.
	}
}

// checkForm checks if the form contains the required fields.
// Every request must contain connection UUID or device ID.
func checkForm(ctx *gin.Context, form interface{}) (string, bool) {
	var base struct {
		Conn   string `json:"uuid" yaml:"uuid" form:"uuid"`
		Device string `json:"device" yaml:"device" form:"device"`
	}
	if form != nil && ctx.ShouldBind(form) != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, modules.Packet{Code: -1, Msg: `${i18n|invalidParameter}`})
		return ``, false
	}
	if ctx.ShouldBind(&base) != nil || (len(base.Conn) == 0 && len(base.Device) == 0) {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, modules.Packet{Code: -1, Msg: `${i18n|invalidParameter}`})
		return ``, false
	}
	connUUID, ok := common.CheckDevice(base.Device, base.Conn)
	if !ok {
		ctx.AbortWithStatusJSON(http.StatusBadGateway, modules.Packet{Code: 1, Msg: `${i18n|deviceNotExists}`})
		return ``, false
	}
	return connUUID, true
}
