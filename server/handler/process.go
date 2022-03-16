package handler

import (
	"Spark/modules"
	"Spark/server/common"
	"Spark/utils"
	"Spark/utils/melody"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"time"
)

// listDeviceProcesses will list processes on remote client
func listDeviceProcesses(ctx *gin.Context) {
	var form struct {
		Conn   string `json:"uuid" yaml:"uuid" form:"uuid"`
		Device string `json:"device" yaml:"device" form:"device"`
	}
	if ctx.ShouldBind(&form) != nil || (len(form.Conn) == 0 && len(form.Device) == 0) {
		ctx.JSON(http.StatusBadRequest, modules.Packet{Code: -1, Msg: `参数不完整`})
		return
	}
	connUUID := ``
	trigger := utils.GetStrUUID()
	if len(form.Conn) == 0 {
		ok := false
		connUUID, ok = common.CheckDevice(form.Device)
		if !ok {
			ctx.JSON(http.StatusBadGateway, modules.Packet{Code: 1, Msg: `未找到该设备`})
			return
		}
	} else {
		connUUID = form.Conn
		if !common.Devices.Has(connUUID) {
			ctx.JSON(http.StatusBadGateway, modules.Packet{Code: 1, Msg: `未找到该设备`})
			return
		}
	}
	common.SendPackUUID(modules.Packet{Act: `listProcesses`, Data: gin.H{`event`: trigger}}, connUUID)
	ok := addEventOnce(func(p modules.Packet, _ *melody.Session) {
		if p.Code != 0 {
			ctx.JSON(http.StatusInternalServerError, modules.Packet{Code: 1, Msg: p.Msg})
		} else {
			ctx.JSON(http.StatusOK, modules.Packet{Code: 0, Data: p.Data})
		}
	}, connUUID, trigger, 5*time.Second)
	if !ok {
		ctx.JSON(http.StatusGatewayTimeout, modules.Packet{Code: 1, Msg: `响应超时`})
	}
}

// killDeviceProcess will try to get send a packet to
// client and let it kill the process specified.
func killDeviceProcess(ctx *gin.Context) {
	var form struct {
		Pid    int32  `json:"pid" yaml:"pid" form:"pid" binding:"required"`
		Conn   string `json:"uuid" yaml:"uuid" form:"uuid"`
		Device string `json:"device" yaml:"device" form:"device"`
	}
	if ctx.ShouldBind(&form) != nil || (len(form.Conn) == 0 && len(form.Device) == 0) {
		ctx.JSON(http.StatusBadRequest, modules.Packet{Code: -1, Msg: `参数不完整`})
		return
	}
	target := ``
	trigger := utils.GetStrUUID()
	if len(form.Conn) == 0 {
		ok := false
		target, ok = common.CheckDevice(form.Device)
		if !ok {
			ctx.JSON(http.StatusBadGateway, modules.Packet{Code: 1, Msg: `未找到该设备`})
			return
		}
	} else {
		target = form.Conn
		if !common.Devices.Has(target) {
			ctx.JSON(http.StatusBadGateway, modules.Packet{Code: 1, Msg: `未找到该设备`})
			return
		}
	}
	common.SendPackUUID(modules.Packet{Code: 0, Act: `killProcess`, Data: gin.H{`pid`: strconv.FormatInt(int64(form.Pid), 10), `event`: trigger}}, target)
	ok := addEventOnce(func(p modules.Packet, _ *melody.Session) {
		if p.Code != 0 {
			ctx.JSON(http.StatusInternalServerError, modules.Packet{Code: 1, Msg: p.Msg})
		} else {
			ctx.JSON(http.StatusOK, modules.Packet{Code: 0})
		}
	}, target, trigger, 5*time.Second)
	if !ok {
		ctx.JSON(http.StatusGatewayTimeout, modules.Packet{Code: 1, Msg: `响应超时`})
	}
}
