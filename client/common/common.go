package common

import (
	"Spark/client/config"
	"Spark/modules"
	"Spark/utils"
	"encoding/binary"
	"encoding/hex"
	"errors"
	ws "github.com/gorilla/websocket"
	"github.com/imroc/req/v3"
	"sync"
	"time"
)

type Conn struct {
	*ws.Conn
	secret    []byte
	secretHex string
}

const MaxMessageSize = (2 << 15) + 1024

var WSConn *Conn
var Mutex = &sync.Mutex{}
var HTTP = CreateClient()

func CreateConn(wsConn *ws.Conn, secret []byte) *Conn {
	return &Conn{
		Conn:      wsConn,
		secret:    secret,
		secretHex: hex.EncodeToString(secret),
	}
}

func CreateClient() *req.Client {
	return req.C().SetUserAgent(`SPARK COMMIT: ` + config.COMMIT)
}

func (wsConn *Conn) SendData(data []byte) error {
	Mutex.Lock()
	defer Mutex.Unlock()
	if WSConn == nil {
		return errors.New(`${i18n|COMMON.DISCONNECTED}`)
	}
	wsConn.SetWriteDeadline(utils.Now.Add(5 * time.Second))
	defer wsConn.SetWriteDeadline(time.Time{})
	return wsConn.WriteMessage(ws.BinaryMessage, data)
}

func (wsConn *Conn) SendPack(pack any) error {
	Mutex.Lock()
	defer Mutex.Unlock()
	data, err := utils.JSON.Marshal(pack)
	if err != nil {
		return err
	}
	data, err = utils.Encrypt(data, wsConn.secret)
	if err != nil {
		return err
	}
	if len(data) > MaxMessageSize {
		_, err = HTTP.R().
			SetBody(data).
			SetHeader(`Secret`, wsConn.secretHex).
			Send(`POST`, config.GetBaseURL(false)+`/ws`)
		return err
	}
	if WSConn == nil {
		return errors.New(`${i18n|COMMON.DISCONNECTED}`)
	}
	wsConn.SetWriteDeadline(utils.Now.Add(5 * time.Second))
	defer wsConn.SetWriteDeadline(time.Time{})
	return wsConn.WriteMessage(ws.BinaryMessage, data)
}

func (wsConn *Conn) SendRawData(event, data []byte, service byte, op byte) error {
	Mutex.Lock()
	defer Mutex.Unlock()
	if WSConn == nil {
		return errors.New(`${i18n|COMMON.DISCONNECTED}`)
	}
	buffer := make([]byte, 24)
	copy(buffer[6:22], event)
	copy(buffer[:4], []byte{34, 22, 19, 17})
	buffer[4] = service
	buffer[5] = op
	binary.BigEndian.PutUint16(buffer[22:24], uint16(len(data)))
	buffer = append(buffer, data...)

	wsConn.SetWriteDeadline(utils.Now.Add(5 * time.Second))
	defer wsConn.SetWriteDeadline(time.Time{})
	return wsConn.WriteMessage(ws.BinaryMessage, buffer)
}

func (wsConn *Conn) SendCallback(pack, prev modules.Packet) error {
	if len(prev.Event) > 0 {
		pack.Event = prev.Event
	}
	return wsConn.SendPack(pack)
}

func (wsConn *Conn) GetSecret() []byte {
	return wsConn.secret
}

func (wsConn *Conn) GetSecretHex() string {
	return wsConn.secretHex
}
