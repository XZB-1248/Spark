package handler

import (
	"Spark/modules"
	"Spark/server/common"
	"Spark/utils"
	"Spark/utils/cmap"
	"Spark/utils/melody"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

type terminal struct {
	session    *melody.Session
	deviceConn *melody.Session
	device     string
	termUUID   string
	eventUUID  string
}

var terminals = cmap.New()
var wsTerminals = melody.New()

func init() {
	wsTerminals.HandleConnect(func(session *melody.Session) {
		device, ok := session.Get(`Device`)
		if !ok {
			simpleSendPack(modules.Packet{Act: `warn`, Msg: `终端创建失败`}, session)
			session.Close()
			return
		}
		val, ok := session.Get(`Terminal`)
		if !ok {
			simpleSendPack(modules.Packet{Act: `warn`, Msg: `终端创建失败`}, session)
			session.Close()
			return
		}
		termUUID, ok := val.(string)
		if !ok {
			simpleSendPack(modules.Packet{Act: `warn`, Msg: `终端创建失败`}, session)
			session.Close()
			return
		}
		connUUID, ok := common.CheckDevice(device.(string))
		if !ok {
			simpleSendPack(modules.Packet{Act: `warn`, Msg: `设备不存在或已经离线`}, session)
			session.Close()
			return
		}
		deviceConn, ok := common.Melody.GetSessionByUUID(connUUID)
		if !ok {
			simpleSendPack(modules.Packet{Act: `warn`, Msg: `设备不存在或已经离线`}, session)
			session.Close()
			return
		}
		eventUUID := utils.GetStrUUID()
		terminal := &terminal{
			session:    session,
			deviceConn: deviceConn,
			device:     device.(string),
			termUUID:   termUUID,
			eventUUID:  eventUUID,
		}
		terminals.Set(termUUID, terminal)
		common.AddEvent(eventWrapper(terminal), connUUID, eventUUID)
		common.SendPack(modules.Packet{Act: `initTerminal`, Data: gin.H{
			`terminal`: termUUID,
		}, Event: eventUUID}, deviceConn)
	})
	wsTerminals.HandleMessage(onMessage)
	wsTerminals.HandleMessageBinary(onMessage)
	wsTerminals.HandleDisconnect(func(session *melody.Session) {
		val, ok := session.Get(`Terminal`)
		if !ok {
			return
		}
		termUUID, ok := val.(string)
		if !ok {
			return
		}
		val, ok = terminals.Get(termUUID)
		if !ok {
			return
		}
		terminal, ok := val.(*terminal)
		if !ok {
			return
		}
		common.SendPack(modules.Packet{Act: `killTerminal`, Data: gin.H{
			`terminal`: terminal.termUUID,
		}, Event: terminal.eventUUID}, terminal.deviceConn)
		terminals.Remove(termUUID)
		common.RemoveEvent(terminal.eventUUID)
	})
	go common.WSHealthCheck(wsTerminals)
}

// initTerminal 负责处理terminal的websocket握手事务
func initTerminal(ctx *gin.Context) {
	if !ctx.IsWebsocket() {
		ctx.Status(http.StatusUpgradeRequired)
		return
	}
	secretStr, ok := ctx.GetQuery(`secret`)
	if !ok || len(secretStr) != 32 {
		ctx.Status(http.StatusBadRequest)
		return
	}
	secret, err := hex.DecodeString(secretStr)
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}
	device, ok := ctx.GetQuery(`device`)
	if !ok {
		ctx.Status(http.StatusBadRequest)
		return
	}
	if _, ok := common.CheckDevice(device); !ok {
		ctx.Status(http.StatusBadRequest)
		return
	}

	wsTerminals.HandleRequestWithKeys(ctx.Writer, ctx.Request, nil, gin.H{
		`Secret`:   secret,
		`Device`:   device,
		`LastPack`: time.Now().Unix(),
		`Terminal`: utils.GetStrUUID(),
	})
}

// eventWrapper 会包装一个eventCb，当收到与浏览器session对应的device响应时，
// 会自动把数据转发给浏览器端
func eventWrapper(terminal *terminal) common.EventCallback {
	return func(pack modules.Packet, device *melody.Session) {
		if pack.Act == `initTerminal` {
			if pack.Code != 0 {
				msg := `终端创建失败：未知错误`
				if len(pack.Msg) > 0 {
					msg = `终端创建失败：` + pack.Msg
				}
				simpleSendPack(modules.Packet{Act: `warn`, Msg: msg}, terminal.session)
				terminals.Remove(terminal.termUUID)
				common.RemoveEvent(terminal.eventUUID)
				terminal.session.Close()
			}
			return
		}
		if pack.Act == `quitTerminal` {
			msg := `终端已退出`
			if len(pack.Msg) > 0 {
				msg = pack.Msg
			}
			simpleSendPack(modules.Packet{Act: `warn`, Msg: msg}, terminal.session)
			terminals.Remove(terminal.termUUID)
			common.RemoveEvent(terminal.eventUUID)
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

func onMessage(session *melody.Session, data []byte) {
	var pack modules.Packet
	data, ok := simpleDecrypt(data, session)
	if !(ok && utils.JSON.Unmarshal(data, &pack) == nil) {
		simpleSendPack(modules.Packet{Code: -1}, session)
		session.Close()
		return
	}
	session.Set(`LastPack`, time.Now().Unix())
	if pack.Act == `inputTerminal` {
		val, ok := session.Get(`Terminal`)
		if !ok {
			return
		}
		termUUID, ok := val.(string)
		if !ok {
			return
		}
		val, ok = terminals.Get(termUUID)
		if !ok {
			return
		}
		terminal, ok := val.(*terminal)
		if !ok {
			return
		}
		if pack.Data == nil {
			return
		}
		if input, ok := pack.Data[`input`]; ok {
			common.SendPack(modules.Packet{Act: `inputTerminal`, Data: gin.H{
				`input`:    input,
				`terminal`: terminal.termUUID,
			}, Event: terminal.eventUUID}, terminal.deviceConn)
		}
	}
}
