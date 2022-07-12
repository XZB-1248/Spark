package common

import (
	"Spark/client/config"
	"Spark/modules"
	"Spark/utils"
	"encoding/hex"
	"errors"
	ws "github.com/gorilla/websocket"
	"github.com/imroc/req/v3"
	"sync"
	"time"
)

type Conn struct {
	*ws.Conn
	Secret []byte
}

var WSConn *Conn
var WSLock = sync.Mutex{}
var HTTP = req.C().SetUserAgent(`SPARK COMMIT: ` + config.COMMIT)

const MaxMessageSize = 32768 + 1024

func SendData(data []byte, wsConn *Conn) error {
	WSLock.Lock()
	defer WSLock.Unlock()
	if WSConn == nil {
		return errors.New(`${i18n|wsClosed}`)
	}
	wsConn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	defer wsConn.SetWriteDeadline(time.Time{})
	return wsConn.WriteMessage(ws.BinaryMessage, data)
}

func SendPack(pack interface{}, wsConn *Conn) error {
	WSLock.Lock()
	defer WSLock.Unlock()
	data, err := utils.JSON.Marshal(pack)
	if err != nil {
		return err
	}
	data, err = utils.Encrypt(data, wsConn.Secret)
	if err != nil {
		return err
	}
	if len(data) > MaxMessageSize {
		_, err = HTTP.R().
			SetBody(data).
			SetHeader(`Secret`, hex.EncodeToString(wsConn.Secret)).
			Send(`POST`, config.GetBaseURL(false)+`/ws`)
		return err
	}
	if WSConn == nil {
		return errors.New(`${i18n|wsClosed}`)
	}
	wsConn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	defer wsConn.SetWriteDeadline(time.Time{})
	return wsConn.WriteMessage(ws.BinaryMessage, data)
}

func SendCb(pack, prev modules.Packet, wsConn *Conn) error {
	if len(prev.Event) > 0 {
		pack.Event = prev.Event
	}
	return SendPack(pack, wsConn)
}
