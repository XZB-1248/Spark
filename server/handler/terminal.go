package handler

import (
	"Spark/modules"
	"Spark/server/common"
	"Spark/utils"
	"Spark/utils/melody"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

type terminal struct {
	uuid       string
	event      string
	device     string
	session    *melody.Session
	deviceConn *melody.Session
}

var wsSessions = melody.New()

func init() {
	wsSessions.HandleConnect(onConnect)
	wsSessions.HandleMessage(onMessage)
	wsSessions.HandleMessageBinary(onMessage)
	wsSessions.HandleDisconnect(onDisconnect)
	go wsHealthCheck(wsSessions)
}

// initTerminal handles terminal websocket handshake event
func initTerminal(ctx *gin.Context) {
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

	wsSessions.HandleRequestWithKeys(ctx.Writer, ctx.Request, nil, gin.H{
		`Secret`:   secret,
		`Device`:   device,
		`LastPack`: common.Unix,
	})
}

// eventWrapper returns a eventCb function that will be called when
// device need to send a packet to browser terminal
func eventWrapper(terminal *terminal) common.EventCallback {
	return func(pack modules.Packet, device *melody.Session) {
		if pack.Act == `initTerminal` {
			if pack.Code != 0 {
				msg := `${i18n|terminalSessionCreationFailed}`
				if len(pack.Msg) > 0 {
					msg += `: ` + pack.Msg
				} else {
					msg += `${i18n|unknownError}`
				}
				simpleSendPack(modules.Packet{Act: `warn`, Msg: msg}, terminal.session)
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
			simpleSendPack(modules.Packet{Act: `warn`, Msg: msg}, terminal.session)
			common.RemoveEvent(terminal.event)
			terminal.session.Close()
			return
		}
		if pack.Act == `outputTerminal` {
			if pack.Data == nil {
				return
			}
			if output, ok := pack.Data[`output`]; ok {
				simpleSendPack(modules.Packet{Act: `outputTerminal`, Data: gin.H{
					`output`: output,
				}}, terminal.session)
			}
		}
	}
}

func wsHealthCheck(container *melody.Melody) {
	const MaxIdleSeconds = 300
	ping := func(uuid string, s *melody.Session) {
		if !simpleSendPack(modules.Packet{Act: `ping`}, s) {
			s.Close()
		}
	}
	for now := range time.NewTicker(60 * time.Second).C {
		timestamp := now.Unix()
		// stores sessions to be disconnected
		queue := make([]*melody.Session, 0)
		container.IterSessions(func(uuid string, s *melody.Session) bool {
			go ping(uuid, s)
			val, ok := s.Get(`LastPack`)
			if !ok {
				queue = append(queue, s)
				return true
			}
			lastPack, ok := val.(int64)
			if !ok {
				queue = append(queue, s)
				return true
			}
			if timestamp-lastPack > MaxIdleSeconds {
				queue = append(queue, s)
			}
			return true
		})
		for i := 0; i < len(queue); i++ {
			queue[i].Close()
		}
	}
}

func onConnect(session *melody.Session) {
	device, ok := session.Get(`Device`)
	if !ok {
		simpleSendPack(modules.Packet{Act: `warn`, Msg: `${i18n|terminalSessionCreationFailed}`}, session)
		session.Close()
		return
	}
	connUUID, ok := common.CheckDevice(device.(string), ``)
	if !ok {
		simpleSendPack(modules.Packet{Act: `warn`, Msg: `${i18n|deviceNotExists}`}, session)
		session.Close()
		return
	}
	deviceConn, ok := common.Melody.GetSessionByUUID(connUUID)
	if !ok {
		simpleSendPack(modules.Packet{Act: `warn`, Msg: `${i18n|deviceNotExists}`}, session)
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
	common.AddEvent(eventWrapper(terminal), connUUID, eventUUID)
	common.SendPack(modules.Packet{Act: `initTerminal`, Data: gin.H{
		`terminal`: termUUID,
	}, Event: eventUUID}, deviceConn)
}

func onMessage(session *melody.Session, data []byte) {
	var pack modules.Packet
	data, ok := simpleDecrypt(data, session)
	if !(ok && utils.JSON.Unmarshal(data, &pack) == nil) {
		simpleSendPack(modules.Packet{Code: -1}, session)
		session.Close()
		return
	}
	session.Set(`LastPack`, common.Unix)
	if pack.Act == `inputTerminal` {
		val, ok := session.Get(`Terminal`)
		if !ok {
			return
		}
		terminal := val.(*terminal)
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
		val, ok := session.Get(`Terminal`)
		if !ok {
			return
		}
		terminal := val.(*terminal)
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
		val, ok := session.Get(`Terminal`)
		if !ok {
			return
		}
		terminal := val.(*terminal)
		if pack.Data == nil {
			return
		}
		common.SendPack(modules.Packet{Act: `killTerminal`, Data: gin.H{
			`terminal`: terminal.uuid,
		}, Event: terminal.event}, terminal.deviceConn)
		return
	}
	if pack.Act == `pong` {
		return
	}
	session.Close()
}

func onDisconnect(session *melody.Session) {
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

func simpleEncrypt(data []byte, session *melody.Session) ([]byte, bool) {
	temp, ok := session.Get(`Secret`)
	if !ok {
		return nil, false
	}
	secret := temp.([]byte)
	block, err := aes.NewCipher(secret)
	if err != nil {
		return nil, false
	}
	stream := cipher.NewCTR(block, secret)
	encBuffer := make([]byte, len(data))
	stream.XORKeyStream(encBuffer, data)
	return encBuffer, true
}

func simpleDecrypt(data []byte, session *melody.Session) ([]byte, bool) {
	temp, ok := session.Get(`Secret`)
	if !ok {
		return nil, false
	}
	secret := temp.([]byte)
	block, err := aes.NewCipher(secret)
	if err != nil {
		return nil, false
	}
	stream := cipher.NewCTR(block, secret)
	decBuffer := make([]byte, len(data))
	stream.XORKeyStream(decBuffer, data)
	return decBuffer, true
}

func simpleSendPack(pack modules.Packet, session *melody.Session) bool {
	if session == nil {
		return false
	}
	data, err := utils.JSON.Marshal(pack)
	if err != nil {
		return false
	}
	data, ok := simpleEncrypt(data, session)
	if !ok {
		return false
	}
	err = session.WriteBinary(data)
	return err == nil
}

func CloseSessionsByDevice(deviceID string) {
	var queue []*melody.Session
	wsSessions.IterSessions(func(_ string, session *melody.Session) bool {
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
