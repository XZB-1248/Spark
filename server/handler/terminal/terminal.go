package terminal

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

type terminal struct {
	uuid       string
	event      string
	device     string
	session    *melody.Session
	deviceConn *melody.Session
}

var terminalSessions = melody.New()

func init() {
	terminalSessions.HandleConnect(onTerminalConnect)
	terminalSessions.HandleMessage(onTerminalMessage)
	terminalSessions.HandleMessageBinary(onTerminalMessage)
	terminalSessions.HandleDisconnect(onTerminalDisconnect)
	go utility.WSHealthCheck(terminalSessions, sendPack)
}

// InitTerminal handles terminal websocket handshake event
func InitTerminal(ctx *gin.Context) {
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

	terminalSessions.HandleRequestWithKeys(ctx.Writer, ctx.Request, nil, gin.H{
		`Secret`:   secret,
		`Device`:   device,
		`LastPack`: common.Unix,
	})
}

// terminalEventWrapper returns a eventCb function that will be called when
// device need to send a packet to browser terminal
func terminalEventWrapper(terminal *terminal) common.EventCallback {
	return func(pack modules.Packet, device *melody.Session) {
		if pack.Act == `initTerminal` {
			if pack.Code != 0 {
				msg := `${i18n|terminalSessionCreationFailed}`
				if len(pack.Msg) > 0 {
					msg += `: ` + pack.Msg
				} else {
					msg += `${i18n|unknownError}`
				}
				sendPack(modules.Packet{Act: `warn`, Msg: msg}, terminal.session)
				common.RemoveEvent(terminal.event)
				terminal.session.Close()
			}
			return
		}
		if pack.Act == `quitTerminal` {
			msg := `${i18n|terminalSessionClosed}`
			if len(pack.Msg) > 0 {
				msg = pack.Msg
			}
			sendPack(modules.Packet{Act: `warn`, Msg: msg}, terminal.session)
			common.RemoveEvent(terminal.event)
			terminal.session.Close()
			return
		}
		if pack.Act == `outputTerminal` {
			if pack.Data == nil {
				return
			}
			if output, ok := pack.Data[`output`]; ok {
				sendPack(modules.Packet{Act: `outputTerminal`, Data: gin.H{
					`output`: output,
				}}, terminal.session)
			}
		}
	}
}

func onTerminalConnect(session *melody.Session) {
	device, ok := session.Get(`Device`)
	if !ok {
		sendPack(modules.Packet{Act: `warn`, Msg: `${i18n|terminalSessionCreationFailed}`}, session)
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
	termUUID := utils.GetStrUUID()
	eventUUID := utils.GetStrUUID()
	terminal := &terminal{
		uuid:       termUUID,
		event:      eventUUID,
		device:     device.(string),
		session:    session,
		deviceConn: deviceConn,
	}
	session.Set(`Terminal`, terminal)
	common.AddEvent(terminalEventWrapper(terminal), connUUID, eventUUID)
	common.SendPack(modules.Packet{Act: `initTerminal`, Data: gin.H{
		`terminal`: termUUID,
	}, Event: eventUUID}, deviceConn)
}

func onTerminalMessage(session *melody.Session, data []byte) {
	var pack modules.Packet
	data, ok := utility.SimpleDecrypt(data, session)
	if !(ok && utils.JSON.Unmarshal(data, &pack) == nil) {
		sendPack(modules.Packet{Code: -1}, session)
		session.Close()
		return
	}
	val, ok := session.Get(`Terminal`)
	if !ok {
		return
	}
	terminal := val.(*terminal)
	session.Set(`LastPack`, common.Unix)
	if pack.Act == `inputTerminal` {
		if pack.Data == nil {
			return
		}
		if input, ok := pack.Data[`input`]; ok {
			common.SendPack(modules.Packet{Act: `inputTerminal`, Data: gin.H{
				`input`:    input,
				`terminal`: terminal.uuid,
			}, Event: terminal.event}, terminal.deviceConn)
		}
		return
	}
	if pack.Act == `resizeTerminal` {
		if pack.Data == nil {
			return
		}
		if width, ok := pack.Data[`width`]; ok {
			if height, ok := pack.Data[`height`]; ok {
				common.SendPack(modules.Packet{Act: `resizeTerminal`, Data: gin.H{
					`width`:    width,
					`height`:   height,
					`terminal`: terminal.uuid,
				}, Event: terminal.event}, terminal.deviceConn)
			}
		}
		return
	}
	if pack.Act == `killTerminal` {
		if pack.Data == nil {
			return
		}
		common.SendPack(modules.Packet{Act: `killTerminal`, Data: gin.H{
			`terminal`: terminal.uuid,
		}, Event: terminal.event}, terminal.deviceConn)
		return
	}
	if pack.Act == `ping` {
		if pack.Data == nil {
			return
		}
		common.SendPack(modules.Packet{Act: `pingTerminal`, Data: gin.H{
			`terminal`: terminal.uuid,
		}, Event: terminal.event}, terminal.deviceConn)
		return
	}
	session.Close()
}

func onTerminalDisconnect(session *melody.Session) {
	val, ok := session.Get(`Terminal`)
	if !ok {
		return
	}
	terminal, ok := val.(*terminal)
	if !ok {
		return
	}
	common.SendPack(modules.Packet{Act: `killTerminal`, Data: gin.H{
		`terminal`: terminal.uuid,
	}, Event: terminal.event}, terminal.deviceConn)
	common.RemoveEvent(terminal.event)
	session.Set(`Terminal`, nil)
	terminal = nil
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
	err = session.WriteBinary(data)
	return err == nil
}

func CloseSessionsByDevice(deviceID string) {
	var queue []*melody.Session
	terminalSessions.IterSessions(func(_ string, session *melody.Session) bool {
		val, ok := session.Get(`Terminal`)
		if !ok {
			return true
		}
		terminal, ok := val.(*terminal)
		if !ok {
			return true
		}
		if terminal.device == deviceID {
			queue = append(queue, session)
			return false
		}
		return true
	})
	for _, session := range queue {
		session.Close()
	}
}
