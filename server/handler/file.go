package handler

import (
	"Spark/modules"
	"Spark/server/common"
	"Spark/utils"
	"Spark/utils/melody"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"
)

// removeDeviceFiles will try to get send a packet to
// client and let it upload the file specified.
func removeDeviceFiles(ctx *gin.Context) {
	var form struct {
		Files []string `json:"files" yaml:"files" form:"files" binding:"required"`
	}
	target, ok := checkForm(ctx, &form)
	if !ok {
		return
	}
	trigger := utils.GetStrUUID()
	common.SendPackByUUID(modules.Packet{Code: 0, Act: `removeFiles`, Data: gin.H{`files`: form.Files}, Event: trigger}, target)
	ok = common.AddEventOnce(func(p modules.Packet, _ *melody.Session) {
		if p.Code != 0 {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, modules.Packet{Code: 1, Msg: p.Msg})
		} else {
			ctx.JSON(http.StatusOK, modules.Packet{Code: 0})
		}
	}, target, trigger, 5*time.Second)
	if !ok {
		ctx.AbortWithStatusJSON(http.StatusGatewayTimeout, modules.Packet{Code: 1, Msg: `${i18n|responseTimeout}`})
	}
}

// listDeviceFiles will list files on remote client
func listDeviceFiles(ctx *gin.Context) {
	var form struct {
		Path string `json:"path" yaml:"path" form:"path" binding:"required"`
	}
	target, ok := checkForm(ctx, &form)
	if !ok {
		return
	}
	trigger := utils.GetStrUUID()
	common.SendPackByUUID(modules.Packet{Act: `listFiles`, Data: gin.H{`path`: form.Path}, Event: trigger}, target)
	ok = common.AddEventOnce(func(p modules.Packet, _ *melody.Session) {
		if p.Code != 0 {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, modules.Packet{Code: 1, Msg: p.Msg})
		} else {
			ctx.JSON(http.StatusOK, modules.Packet{Code: 0, Data: p.Data})
		}
	}, target, trigger, 5*time.Second)
	if !ok {
		ctx.AbortWithStatusJSON(http.StatusGatewayTimeout, modules.Packet{Code: 1, Msg: `${i18n|responseTimeout}`})
	}
}

// getDeviceFiles will try to get send a packet to
// client and let it upload the file specified.
func getDeviceFiles(ctx *gin.Context) {
	var form struct {
		Files   []string `json:"files" yaml:"files" form:"files" binding:"required"`
		Preview bool     `json:"preview" yaml:"preview" form:"preview"`
	}
	target, ok := checkForm(ctx, &form)
	if !ok {
		return
	}
	if len(form.Files) == 0 {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, modules.Packet{Code: -1, Msg: `${i18n|invalidParameter}`})
		return
	}
	bridgeID := utils.GetStrUUID()
	trigger := utils.GetStrUUID()
	var rangeStart, rangeEnd int64
	var err error
	partial := false
	{
		command := gin.H{`files`: form.Files, `bridge`: bridgeID}
		rangeHeader := ctx.GetHeader(`Range`)
		if len(rangeHeader) > 6 {
			if rangeHeader[:6] != `bytes=` {
				ctx.AbortWithStatus(http.StatusRequestedRangeNotSatisfiable)
				return
			}
			rangeHeader = strings.TrimSpace(rangeHeader[6:])
			rangesList := strings.Split(rangeHeader, `,`)
			if len(rangesList) > 1 {
				ctx.AbortWithStatus(http.StatusRequestedRangeNotSatisfiable)
				return
			}
			r := strings.Split(rangesList[0], `-`)
			rangeStart, err = strconv.ParseInt(r[0], 10, 64)
			if err != nil {
				ctx.AbortWithStatus(http.StatusRequestedRangeNotSatisfiable)
				return
			}
			if len(r[1]) > 0 {
				rangeEnd, err = strconv.ParseInt(r[1], 10, 64)
				if err != nil {
					ctx.AbortWithStatus(http.StatusRequestedRangeNotSatisfiable)
					return
				}
				if rangeEnd < rangeStart {
					ctx.AbortWithStatus(http.StatusRequestedRangeNotSatisfiable)
					return
				}
				command[`end`] = rangeEnd
			}
			command[`start`] = rangeStart
			partial = true
		}
		common.SendPackByUUID(modules.Packet{Code: 0, Act: `uploadFiles`, Data: command, Event: trigger}, target)
	}
	wait := make(chan bool)
	called := false
	common.AddEvent(func(p modules.Packet, _ *melody.Session) {
		wait <- false
		called = true
		removeBridge(bridgeID)
		common.RemoveEvent(trigger)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, modules.Packet{Code: 1, Msg: p.Msg})
	}, target, trigger)
	instance := addBridgeWithDst(nil, bridgeID, ctx)
	instance.OnPush = func(bridge *bridge) {
		called = true
		common.RemoveEvent(trigger)
		src := bridge.src
		for k, v := range src.Request.Header {
			if strings.HasPrefix(k, `File`) {
				ctx.Header(k, v[0])
			}
		}
		if !form.Preview {
			if len(form.Files) == 1 {
				ctx.Header(`Accept-Ranges`, `bytes`)
				if src.Request.ContentLength > 0 {
					ctx.Header(`Content-Length`, strconv.FormatInt(src.Request.ContentLength, 10))
				}
			} else {
				ctx.Header(`Accept-Ranges`, `none`)
			}
			ctx.Header(`Content-Transfer-Encoding`, `binary`)
			ctx.Header(`Content-Type`, `application/octet-stream`)
			filename := src.GetHeader(`FileName`)
			if len(filename) == 0 {
				if len(form.Files) > 1 {
					filename = `Archive.zip`
				} else {
					filename = path.Base(strings.ReplaceAll(form.Files[0], `\`, `/`))
				}
			}
			ctx.Header(`Content-Disposition`, fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, filename, filename))
		}

		if partial {
			if rangeEnd == 0 {
				rangeEnd, err = strconv.ParseInt(src.GetHeader(`FileSize`), 10, 64)
				if err == nil {
					ctx.Header(`Content-Range`, fmt.Sprintf(`bytes %d-%d/%d`, rangeStart, rangeEnd-1, rangeEnd))
				}
			} else {
				ctx.Header(`Content-Range`, fmt.Sprintf(`bytes %d-%d/%v`, rangeStart, rangeEnd, src.GetHeader(`FileSize`)))
			}
			ctx.Status(http.StatusPartialContent)
		} else {
			ctx.Status(http.StatusOK)
		}
	}
	instance.OnFinish = func(bridge *bridge) {
		wait <- false
	}
	select {
	case <-wait:
	case <-time.After(5 * time.Second):
		if !called {
			removeBridge(bridgeID)
			common.RemoveEvent(trigger)
			ctx.AbortWithStatusJSON(http.StatusGatewayTimeout, modules.Packet{Code: 1, Msg: `${i18n|responseTimeout}`})
		} else {
			<-wait
		}
	}
	close(wait)
}

// getDeviceTextFile will try to get send a packet to
// client and let it upload the text file.
func getDeviceTextFile(ctx *gin.Context) {
	var form struct {
		File string `json:"file" yaml:"file" form:"file" binding:"required"`
	}
	target, ok := checkForm(ctx, &form)
	if !ok {
		return
	}
	bridgeID := utils.GetStrUUID()
	trigger := utils.GetStrUUID()
	common.SendPackByUUID(modules.Packet{Code: 0, Act: `uploadTextFile`, Data: gin.H{
		`file`:   form.File,
		`bridge`: bridgeID,
	}, Event: trigger}, target)
	wait := make(chan bool)
	called := false
	common.AddEvent(func(p modules.Packet, _ *melody.Session) {
		wait <- false
		called = true
		removeBridge(bridgeID)
		common.RemoveEvent(trigger)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, modules.Packet{Code: 1, Msg: p.Msg})
	}, target, trigger)
	instance := addBridgeWithDst(nil, bridgeID, ctx)
	instance.OnPush = func(bridge *bridge) {
		called = true
		common.RemoveEvent(trigger)
		src := bridge.src
		for k, v := range src.Request.Header {
			if strings.HasPrefix(k, `File`) {
				ctx.Header(k, v[0])
			}
		}
		ctx.Header(`Accept-Ranges`, `none`)
		ctx.Header(`Content-Transfer-Encoding`, `binary`)
		ctx.Header(`Content-Type`, `application/octet-stream`)
		filename := src.GetHeader(`FileName`)
		if len(filename) == 0 {
			filename = path.Base(strings.ReplaceAll(form.File, `\`, `/`))
		}
		ctx.Header(`Content-Disposition`, fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, filename, filename))
		ctx.Status(http.StatusOK)
	}
	instance.OnFinish = func(bridge *bridge) {
		wait <- false
	}
	select {
	case <-wait:
	case <-time.After(5 * time.Second):
		if !called {
			removeBridge(bridgeID)
			common.RemoveEvent(trigger)
			ctx.AbortWithStatusJSON(http.StatusGatewayTimeout, modules.Packet{Code: 1, Msg: `${i18n|responseTimeout}`})
		} else {
			<-wait
		}
	}
	close(wait)
}

// uploadToDevice handles file from browser
// and transfer it to device.
func uploadToDevice(ctx *gin.Context) {
	var form struct {
		Path string `json:"path" yaml:"path" form:"path" binding:"required"`
		File string `json:"file" yaml:"file" form:"file" binding:"required"`
	}
	target, ok := checkForm(ctx, &form)
	if !ok {
		return
	}
	bridgeID := utils.GetStrUUID()
	trigger := utils.GetStrUUID()
	wait := make(chan bool)
	called := false
	common.AddEvent(func(p modules.Packet, _ *melody.Session) {
		wait <- false
		called = true
		removeBridge(bridgeID)
		common.RemoveEvent(trigger)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, modules.Packet{Code: 1, Msg: p.Msg})
	}, target, trigger)
	instance := addBridgeWithSrc(nil, bridgeID, ctx)
	instance.OnPull = func(bridge *bridge) {
		called = true
		common.RemoveEvent(trigger)
		dst := bridge.dst
		if ctx.Request.ContentLength > 0 {
			dst.Header(`Content-Length`, strconv.FormatInt(ctx.Request.ContentLength, 10))
		}
		dst.Header(`Accept-Ranges`, `none`)
		dst.Header(`Content-Transfer-Encoding`, `binary`)
		dst.Header(`Content-Type`, `application/octet-stream`)
		dst.Header(`Content-Disposition`, fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, form.File, form.File))
	}
	instance.OnFinish = func(bridge *bridge) {
		wait <- false
	}
	common.SendPackByUUID(modules.Packet{Code: 0, Act: `fetchFile`, Data: gin.H{
		`path`:   form.Path,
		`file`:   form.File,
		`bridge`: bridgeID,
	}, Event: trigger}, target)
	select {
	case <-wait:
		ctx.JSON(http.StatusOK, modules.Packet{Code: 0})
	case <-time.After(5 * time.Second):
		if !called {
			removeBridge(bridgeID)
			common.RemoveEvent(trigger)
			ctx.AbortWithStatusJSON(http.StatusGatewayTimeout, modules.Packet{Code: 1, Msg: `${i18n|responseTimeout}`})
		} else {
			<-wait
			ctx.JSON(http.StatusOK, modules.Packet{Code: 0})
		}
	}
	close(wait)
}
