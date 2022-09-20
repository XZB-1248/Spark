package desktop

import (
	"Spark/modules"
	"Spark/server/common"
	"Spark/server/handler/utility"
	"Spark/utils"
	"Spark/utils/melody"
	"encoding/hex"
	"github.com/gin-gonic/gin"
	"net/http"
)

type desktop struct {
	uuid       string
	event      string
	device     string
	targetConn *melody.Session
	deviceConn *melody.Session
}

var desktopSessions = melody.New()

func init() {
	desktopSessions.Config.MaxMessageSize = 32768 + 1024
	desktopSessions.HandleConnect(onDesktopConnect)
	desktopSessions.HandleMessage(onDesktopMessage)
	desktopSessions.HandleMessageBinary(onDesktopMessage)
	desktopSessions.HandleDisconnect(onDesktopDisconnect)
	go utility.WSHealthCheck(desktopSessions, sendPack)
}

// InitDesktop handles desktop websocket handshake event
func InitDesktop(ctx *gin.Context) {
	if !ctx.IsWebsocket() {
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}
	secretStr, ok := ctx.GetQuery(`secret`)
	if !ok || len(secretStr) != 32 {
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}
	secret, err := hex.DecodeString(secretStr)
	if err != nil {
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}
	device, ok := ctx.GetQuery(`device`)
	if !ok {
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if _, ok := common.CheckDevice(device, ``); !ok {
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	desktopSessions.HandleRequestWithKeys(ctx.Writer, ctx.Request, nil, gin.H{
		`Secret`:   secret,
		`Device`:   device,
		`LastPack`: common.Unix,
	})
}

// desktopEventWrapper returns a eventCb function that will be called when
// device need to send a packet to browser
func desktopEventWrapper(desktop *desktop) common.EventCallback {
	return func(pack modules.Packet, device *melody.Session) {
		if len(pack.Act) == 0 {
			if pack.Data == nil {
				return
			}
			if data, ok := pack.Data[`data`]; ok {
				desktop.targetConn.WriteBinary(*data.(*[]byte))
			}
			return
		}
		if pack.Act == `initDesktop` {
			if pack.Code != 0 {
				msg := `${i18n|desktopSessionCreationFailed}`
				if len(pack.Msg) > 0 {
					msg += `: ` + pack.Msg
				} else {
					msg += `${i18n|unknownError}`
				}
				sendPack(modules.Packet{Act: `quit`, Msg: msg}, desktop.targetConn)
				common.RemoveEvent(desktop.event)
				desktop.targetConn.Close()
			}
			return
		}
		if pack.Act == `quitDesktop` {
			msg := `${i18n|desktopSessionClosed}`
			if len(pack.Msg) > 0 {
				msg = pack.Msg
			}
			sendPack(modules.Packet{Act: `quit`, Msg: msg}, desktop.targetConn)
			common.RemoveEvent(desktop.event)
			desktop.targetConn.Close()
			return
		}
	}
}

func onDesktopConnect(session *melody.Session) {
	device, ok := session.Get(`Device`)
	if !ok {
		sendPack(modules.Packet{Act: `warn`, Msg: `${i18n|desktopSessionCreationFailed}`}, session)
		session.Close()
		return
	}
	connUUID, ok := common.CheckDevice(device.(string), ``)
	if !ok {
		sendPack(modules.Packet{Act: `warn`, Msg: `${i18n|deviceNotExists}`}, session)
		session.Close()
		return
	}
	deviceConn, ok := common.Melody.GetSessionByUUID(connUUID)
	if !ok {
		sendPack(modules.Packet{Act: `warn`, Msg: `${i18n|deviceNotExists}`}, session)
		session.Close()
		return
	}
	eventUUID := utils.GetStrUUID()
	desktopUUID := utils.GetStrUUID()
	desktop := &desktop{
		uuid:       desktopUUID,
		event:      eventUUID,
		device:     device.(string),
		targetConn: session,
		deviceConn: deviceConn,
	}
	session.Set(`Desktop`, desktop)
	common.AddEvent(desktopEventWrapper(desktop), connUUID, eventUUID)
	common.SendPack(modules.Packet{Act: `initDesktop`, Data: gin.H{
		`desktop`: desktopUUID,
	}, Event: eventUUID}, deviceConn)
}

func onDesktopMessage(session *melody.Session, data []byte) {
	var pack modules.Packet
	data, ok := utility.SimpleDecrypt(data, session)
	if !(ok && utils.JSON.Unmarshal(data, &pack) == nil) {
		if val, ok := session.Get(`Desktop`); !ok {
			desktop := val.(*desktop)
			common.SendPack(modules.Packet{Act: `killDesktop`, Data: gin.H{
				`desktop`: desktop.uuid,
			}, Event: desktop.event}, desktop.deviceConn)
		}
		sendPack(modules.Packet{Code: -1}, session)
		session.Close()
		return
	}
	val, ok := session.Get(`Desktop`)
	if !ok {
		return
	}
	desktop := val.(*desktop)
	session.Set(`LastPack`, common.Unix)
	if pack.Act == `pingDesktop` {
		common.SendPack(modules.Packet{Act: `pingDesktop`, Data: gin.H{
			`desktop`: desktop.uuid,
		}, Event: desktop.event}, desktop.deviceConn)
		return
	}
	if pack.Act == `killDesktop` {
		common.SendPack(modules.Packet{Act: `killDesktop`, Data: gin.H{
			`desktop`: desktop.uuid,
		}, Event: desktop.event}, desktop.deviceConn)
		return
	}
	if pack.Act == `getDesktop` {
		common.SendPack(modules.Packet{Act: `getDesktop`, Data: gin.H{
			`desktop`: desktop.uuid,
		}, Event: desktop.event}, desktop.deviceConn)
		return
	}
	session.Close()
}

func onDesktopDisconnect(session *melody.Session) {
	val, ok := session.Get(`Desktop`)
	if !ok {
		return
	}
	desktop, ok := val.(*desktop)
	if !ok {
		return
	}
	common.SendPack(modules.Packet{Act: `killDesktop`, Data: gin.H{
		`desktop`: desktop.uuid,
	}, Event: desktop.event}, desktop.deviceConn)
	common.RemoveEvent(desktop.event)
	session.Set(`Desktop`, nil)
	desktop = nil
}

func sendPack(pack modules.Packet, session *melody.Session) bool {
	if session == nil {
		return false
	}
	data, err := utils.JSON.Marshal(pack)
	if err != nil {
		return false
	}
	data, ok := utility.SimpleEncrypt(data, session)
	if !ok {
		return false
	}
	err = session.WriteBinary(append([]byte{00, 22, 34, 19, 20, 03}, data...))
	return err == nil
}

func CloseSessionsByDevice(deviceID string) {
	var queue []*melody.Session
	desktopSessions.IterSessions(func(_ string, session *melody.Session) bool {
		val, ok := session.Get(`Desktop`)
		if !ok {
			return true
		}
		desktop, ok := val.(*desktop)
		if !ok {
			return true
		}
		if desktop.device == deviceID {
			sendPack(modules.Packet{Act: `quit`, Msg: `${i18n|desktopSessionClosed}`}, desktop.targetConn)
			queue = append(queue, session)
			return false
		}
		return true
	})
	for _, session := range queue {
		session.Close()
	}
}
