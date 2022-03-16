package handler

import (
	"Spark/modules"
	"Spark/server/common"
	"Spark/utils"
	"Spark/utils/melody"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/kataras/golog"
	"net/http"
	"time"
)

// APIRouter 负责分配各种API接口
func APIRouter(ctx *gin.RouterGroup, auth gin.HandlerFunc) {
	ctx.PUT(`/device/screenshot/put`, putScreenshot)
	ctx.PUT(`/device/file/put`, putDeviceFile)
	ctx.Any(`/device/terminal`, initTerminal)
	group := ctx.Group(`/`, auth)
	{
		group.POST(`/device/screenshot/get`, getScreenshot)
		group.POST(`/device/process/list`, listDeviceProcesses)
		group.POST(`/device/process/kill`, killDeviceProcess)
		group.POST(`/device/file/remove`, removeDeviceFile)
		group.POST(`/device/file/list`, listDeviceFiles)
		group.POST(`/device/file/get`, getDeviceFile)
		group.POST(`/device/list`, getDevices)
		group.POST(`/device/:act`, callDevice)
		group.POST(`/client/check`, checkClient)
		group.POST(`/client/generate`, generateClient)
	}
}

// putScreenshot 负责获取client发送过来的屏幕截图
func putScreenshot(ctx *gin.Context) {
	errMsg := ctx.GetHeader(`Error`)
	trigger := ctx.GetHeader(`Trigger`)
	if len(trigger) == 0 {
		ctx.JSON(http.StatusBadRequest, modules.Packet{Code: -1, Msg: `参数不完整`})
		return
	}
	if len(errMsg) > 0 {
		evCaller(modules.Packet{
			Code: 1,
			Msg:  fmt.Sprintf(`截图失败：%v`, errMsg),
			Data: map[string]interface{}{
				`callback`: trigger,
			},
		}, nil)
		ctx.JSON(http.StatusOK, modules.Packet{Code: 0})
		return
	}
	data, err := ctx.GetRawData()
	if len(data) == 0 {
		msg := ``
		if err != nil {
			msg = fmt.Sprintf(`截图读取失败：%v`, err)
			ctx.JSON(http.StatusInternalServerError, modules.Packet{Code: 1, Msg: msg})
		} else {
			msg = `截图失败：未知错误`
			ctx.JSON(http.StatusOK, modules.Packet{Code: 0})
		}
		evCaller(modules.Packet{
			Code: 1,
			Msg:  msg,
			Data: map[string]interface{}{
				`callback`: trigger,
			},
		}, nil)
		return
	}
	evCaller(modules.Packet{
		Code: 0,
		Data: map[string]interface{}{
			`screenshot`: data,
			`callback`:   trigger,
		},
	}, nil)
	ctx.JSON(http.StatusOK, modules.Packet{Code: 0})
}

// getScreenshot 负责发送指令给client，让其截图
func getScreenshot(ctx *gin.Context) {
	var form struct {
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
	common.SendPackUUID(modules.Packet{Code: 0, Act: `screenshot`, Data: gin.H{`event`: trigger}}, target)
	ok := addEventOnce(func(p modules.Packet, _ *melody.Session) {
		if p.Code != 0 {
			ctx.JSON(http.StatusInternalServerError, modules.Packet{Code: 1, Msg: p.Msg})
		} else {
			data, ok := p.Data[`screenshot`]
			if !ok {
				ctx.JSON(http.StatusInternalServerError, modules.Packet{Code: 1, Msg: `截图获取失败`})
				return
			}
			screenshot, ok := data.([]byte)
			if !ok {
				ctx.JSON(http.StatusInternalServerError, modules.Packet{Code: 1, Msg: `截图获取失败`})
				return
			}
			ctx.Data(200, `image/png`, screenshot)
		}
	}, target, trigger, 5*time.Second)
	if !ok {
		ctx.JSON(http.StatusGatewayTimeout, modules.Packet{Code: 1, Msg: `响应超时`})
	}
}

// getDevices 负责获取所有device的基本信息
func getDevices(ctx *gin.Context) {
	devices := make(map[string]modules.Device)
	common.Devices.IterCb(func(uuid string, v interface{}) bool {
		device, ok := v.(*modules.Device)
		if ok {
			devices[uuid] = *device
		}
		return true
	})
	ctx.JSON(http.StatusOK, modules.CommonPack{Code: 0, Data: devices})
}

// callDevice 负责把HTTP网关发送的请求转发给client
func callDevice(ctx *gin.Context) {
	var form struct {
		Conn   string `json:"uuid" yaml:"uuid" form:"uuid"`
		Device string `json:"device" yaml:"device" form:"device"`
	}
	act := ctx.Param(`act`)
	if ctx.ShouldBind(&form) != nil || len(act) == 0 || (len(form.Conn) == 0 && len(form.Device) == 0) {
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
	common.SendPackUUID(modules.Packet{Act: act, Data: gin.H{`event`: trigger}}, connUUID)
	ok := addEventOnce(func(p modules.Packet, _ *melody.Session) {
		if p.Code != 0 {
			ctx.JSON(http.StatusInternalServerError, modules.Packet{Code: 1, Msg: p.Msg})
		} else {
			ctx.JSON(http.StatusOK, modules.Packet{Code: 0})
		}
	}, connUUID, trigger, 5*time.Second)
	if !ok {
		ctx.JSON(http.StatusGatewayTimeout, modules.Packet{Code: 1, Msg: `响应超时`})
	}
}

// WSDevice 负责处理client设备信息上报的事件
func WSDevice(data []byte, session *melody.Session) error {
	var pack struct {
		Code   int            `json:"code,omitempty"`
		Act    string         `json:"act,omitempty"`
		Msg    string         `json:"msg,omitempty"`
		Device modules.Device `json:"data"`
	}
	err := utils.JSON.Unmarshal(data, &pack)
	if err != nil {
		golog.Error(err)
		session.Close()
		return err
	}

	addr, ok := session.Get(`Address`)
	if ok {
		pack.Device.WAN = addr.(string)
	} else {
		pack.Device.WAN = `Unknown`
	}

	if pack.Act == `report` {
		// 查询设备列表中，该设备是否已经上线
		// 如果已经上线，就找到对应的session，发送命令使其退出
		exSession := ``
		common.Devices.IterCb(func(uuid string, v interface{}) bool {
			device := v.(*modules.Device)
			if device.ID == pack.Device.ID {
				exSession = uuid
				target, ok := common.Melody.GetSessionByUUID(uuid)
				if ok {
					common.SendPack(modules.Packet{Act: `offline`}, target)
					target.Close()
				}
				return false
			}
			return true
		})
		if len(exSession) > 0 {
			common.Devices.Remove(exSession)
		}
	}

	common.Devices.Set(session.UUID, &pack.Device)
	if pack.Act == `setDevice` {
		common.SendPack(modules.Packet{Act: `heartbeat`}, session)
	} else {
		common.SendPack(modules.Packet{Code: 0}, session)
	}
	return nil
}

// WSRouter 负责处理client回复的packet
func WSRouter(pack modules.Packet, session *melody.Session) {

	evCaller(pack, session)
}
