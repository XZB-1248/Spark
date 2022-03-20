package common

import (
	"Spark/client/config"
	"Spark/modules"
	"Spark/utils"
	"encoding/hex"
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
var lock = sync.Mutex{}

func SendPack(pack interface{}, wsConn *Conn) error {
	lock.Lock()
	defer lock.Unlock()
	data, err := utils.JSON.Marshal(pack)
	if err != nil {
		return err
	}
	data, err = utils.Encrypt(data, wsConn.Secret)
	if err != nil {
		return err
	}
	if len(data) > 1024 {
		_, err = req.R().
			SetBody(data).
			SetHeader(`Secret`, hex.EncodeToString(wsConn.Secret)).
			Send(`POST`, config.GetBaseURL(false)+`/ws`)
		return err
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
