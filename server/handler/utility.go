package handler

import (
	"Spark/modules"
	"Spark/server/common"
	"Spark/server/config"
	"Spark/utils"
	"Spark/utils/melody"
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/kataras/golog"
	"net/http"
	"strconv"
	"time"
)

// OnDevicePack handles events about device info.
// Such as websocket handshake and update device info.
func OnDevicePack(data []byte, session *melody.Session) error {
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
		// Check if this device has already connected.
		// If so, then find the session and let client quit.
		// This will keep only one connection remained per device.
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
	common.SendPack(modules.Packet{Code: 0}, session)

	{
		val, ok := common.Devices.Get(session.UUID)
		if ok {
			deviceInfo := val.(*modules.Device)
			deviceInfo.CPU = pack.Device.CPU
			deviceInfo.RAM = pack.Device.RAM
			deviceInfo.Net = pack.Device.Net
			if pack.Device.Disk.Total > 0 {
				deviceInfo.Disk = pack.Device.Disk
			}
			deviceInfo.Uptime = pack.Device.Uptime
			return nil
		}
		common.Devices.Set(session.UUID, &pack.Device)
	}
	return nil
}

// checkUpdate will check if client need update and return latest client if so.
func checkUpdate(ctx *gin.Context) {
	var form struct {
		OS     string `form:"os" binding:"required"`
		Arch   string `form:"arch" binding:"required"`
		Commit string `form:"commit" binding:"required"`
	}
	if err := ctx.ShouldBind(&form); err != nil {
		golog.Error(err)
		ctx.JSON(http.StatusBadRequest, modules.Packet{Code: -1, Msg: `${i18n|invalidParameter}`})
		return
	}
	if form.Commit == config.COMMIT {
		ctx.JSON(http.StatusOK, modules.Packet{Code: 0})
		return
	}
	tpl, err := common.BuiltFS.Open(fmt.Sprintf(`/%v_%v`, form.OS, form.Arch))
	if err != nil {
		ctx.JSON(http.StatusNotFound, modules.Packet{Code: 1, Msg: `${i18n|osOrArchNotPrebuilt}`})
		return
	}

	const MaxBodySize = 384 // This is size of client config buffer.
	if ctx.Request.ContentLength > MaxBodySize {
		ctx.JSON(http.StatusRequestEntityTooLarge, modules.Packet{Code: 1})
		return
	}
	body, err := ctx.GetRawData()
	if err != nil {
		ctx.JSON(http.StatusBadRequest, modules.Packet{Code: 1})
		return
	}
	session := common.CheckClientReq(ctx)
	if session == nil {
		ctx.JSON(http.StatusUnauthorized, modules.Packet{Code: 1})
		return
	}

	ctx.Header(`Accept-Ranges`, `none`)
	ctx.Header(`Content-Transfer-Encoding`, `binary`)
	ctx.Header(`Content-Type`, `application/octet-stream`)
	if stat, err := tpl.Stat(); err == nil {
		ctx.Header(`Content-Length`, strconv.FormatInt(stat.Size(), 10))
	}
	cfgBuffer := bytes.Repeat([]byte{'\x19'}, 384)
	prevBuffer := make([]byte, 0)
	for {
		thisBuffer := make([]byte, 1024)
		n, err := tpl.Read(thisBuffer)
		thisBuffer = thisBuffer[:n]
		tempBuffer := append(prevBuffer, thisBuffer...)
		bufIndex := bytes.Index(tempBuffer, cfgBuffer)
		if bufIndex > -1 {
			tempBuffer = bytes.Replace(tempBuffer, cfgBuffer, body, -1)
		}
		ctx.Writer.Write(tempBuffer[:len(prevBuffer)])
		prevBuffer = tempBuffer[len(prevBuffer):]
		if err != nil {
			break
		}
	}
	if len(prevBuffer) > 0 {
		ctx.Writer.Write(prevBuffer)
		prevBuffer = []byte{}
	}
}

// getDevices will return all info about all clients.
func getDevices(ctx *gin.Context) {
	devices := map[string]interface{}{}
	common.Devices.IterCb(func(uuid string, v interface{}) bool {
		device := v.(*modules.Device)
		devices[uuid] = *device
		return true
	})
	ctx.JSON(http.StatusOK, modules.Packet{Code: 0, Data: devices})
}

// callDevice will call client with command from browser.
func callDevice(ctx *gin.Context) {
	act := ctx.Param(`act`)
	if len(act) == 0 {
		ctx.JSON(http.StatusBadRequest, modules.Packet{Code: -1, Msg: `${i18n|invalidParameter}`})
		return
	}
	{
		actions := []string{`lock`, `logoff`, `hibernate`, `suspend`, `restart`, `shutdown`, `offline`}
		ok := false
		for _, v := range actions {
			if v == act {
				ok = true
				break
			}
		}
		if !ok {
			ctx.JSON(http.StatusBadRequest, modules.Packet{Code: -1, Msg: `${i18n|invalidParameter}`})
			return
		}
	}
	connUUID, ok := checkForm(ctx, nil)
	if !ok {
		return
	}
	trigger := utils.GetStrUUID()
	common.SendPackByUUID(modules.Packet{Act: act, Event: trigger}, connUUID)
	ok = common.AddEventOnce(func(p modules.Packet, _ *melody.Session) {
		if p.Code != 0 {
			ctx.JSON(http.StatusInternalServerError, modules.Packet{Code: 1, Msg: p.Msg})
		} else {
			ctx.JSON(http.StatusOK, modules.Packet{Code: 0})
		}
	}, connUUID, trigger, 5*time.Second)
	if !ok {
		//This means the client is offline.
		//So we take this as a success.
		ctx.JSON(http.StatusOK, modules.Packet{Code: 0})
	}
}
