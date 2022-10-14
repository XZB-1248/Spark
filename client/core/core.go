package core

import (
	"Spark/client/common"
	"Spark/client/config"
	"Spark/modules"
	"Spark/utils"
	"encoding/hex"
	"errors"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	ws "github.com/gorilla/websocket"
	"github.com/kataras/golog"
)

// simplified type of map
type smap map[string]any

var stop bool
var (
	errNoSecretHeader = errors.New(`can not find secret header`)
)

func Start() {
	for !stop {
		var err error
		if common.WSConn != nil {
			common.Mutex.Lock()
			common.WSConn.Close()
			common.Mutex.Unlock()
		}
		common.Mutex.Lock()
		common.WSConn, err = connectWS()
		common.Mutex.Unlock()
		if err != nil && !stop {
			golog.Error(`Connection error: `, err)
			<-time.After(3 * time.Second)
			continue
		}

		err = reportWS(common.WSConn)
		if err != nil && !stop {
			golog.Error(`Register error: `, err)
			<-time.After(3 * time.Second)
			continue
		}

		checkUpdate(common.WSConn)

		err = handleWS(common.WSConn)
		if err != nil && !stop {
			golog.Error(`Execution error: `, err)
			<-time.After(3 * time.Second)
			continue
		}
	}
}

func connectWS() (*common.Conn, error) {
	wsConn, wsResp, err := ws.DefaultDialer.Dial(config.GetBaseURL(true)+`/ws`, http.Header{
		`UUID`: []string{config.Config.UUID},
		`Key`:  []string{config.Config.Key},
	})
	if err != nil {
		return nil, err
	}
	header, find := wsResp.Header[`Secret`]
	if !find || len(header) == 0 {
		return nil, errNoSecretHeader
	}
	secret, err := hex.DecodeString(header[0])
	if err != nil {
		return nil, err
	}
	return common.CreateConn(wsConn, secret), nil
}

func reportWS(wsConn *common.Conn) error {
	device, err := GetDevice()
	if err != nil {
		return err
	}
	pack := modules.CommonPack{Act: `report`, Data: *device}
	err = wsConn.SendPack(pack)
	common.WSConn.SetWriteDeadline(time.Time{})
	if err != nil {
		return err
	}
	common.WSConn.SetReadDeadline(utils.Now.Add(5 * time.Second))
	_, data, err := common.WSConn.ReadMessage()
	common.WSConn.SetReadDeadline(time.Time{})
	if err != nil {
		return err
	}
	data, err = utils.Decrypt(data, common.WSConn.GetSecret())
	if err != nil {
		return err
	}
	err = utils.JSON.Unmarshal(data, &pack)
	if err != nil {
		return err
	}
	if pack.Code != 0 {
		return errors.New(`${i18n|COMMON.UNKNOWN_ERROR}`)
	}
	return nil
}

func checkUpdate(wsConn *common.Conn) error {
	if len(config.COMMIT) == 0 {
		return nil
	}
	resp, err := common.HTTP.R().
		SetBody(config.ConfigBuffer).
		SetQueryParam(`os`, runtime.GOOS).
		SetQueryParam(`arch`, runtime.GOARCH).
		SetQueryParam(`commit`, config.COMMIT).
		SetHeader(`Secret`, wsConn.GetSecretHex()).
		Send(`POST`, config.GetBaseURL(false)+`/api/client/update`)
	if err != nil {
		return err
	}
	if resp == nil {
		return errors.New(`${i18n|COMMON.UNKNOWN_ERROR}`)
	}
	if strings.HasPrefix(resp.GetContentType(), `application/octet-stream`) {
		body := resp.Bytes()
		if len(body) > 0 {
			selfPath, err := os.Executable()
			if err != nil {
				selfPath = os.Args[0]
			}
			err = os.WriteFile(selfPath+`.tmp`, body, 0755)
			if err != nil {
				return err
			}
			cmd := exec.Command(selfPath+`.tmp`, `--update`)
			err = cmd.Start()
			if err != nil {
				return err
			}
			stop = true
			wsConn.Close()
			os.Exit(0)
		}
		return nil
	}
	return nil
}

func handleWS(wsConn *common.Conn) error {
	errCount := 0
	for {
		_, data, err := wsConn.ReadMessage()
		if err != nil {
			golog.Error(err)
			return nil
		}
		data, err = utils.Decrypt(data, wsConn.GetSecret())
		if err != nil {
			golog.Error(err)
			errCount++
			if errCount > 3 {
				break
			}
			continue
		}
		pack := modules.Packet{}
		utils.JSON.Unmarshal(data, &pack)
		if err != nil {
			golog.Error(err)
			errCount++
			if errCount > 3 {
				break
			}
			continue
		}
		errCount = 0
		if pack.Data == nil {
			pack.Data = smap{}
		}
		go handleAct(pack, wsConn)
	}
	wsConn.Close()
	return nil
}

func handleAct(pack modules.Packet, wsConn *common.Conn) {
	if act, ok := handlers[pack.Act]; !ok {
		wsConn.SendCallback(modules.Packet{Code: 1, Msg: `${i18n|COMMON.OPERATION_NOT_SUPPORTED}`}, pack)
	} else {
		defer func() {
			if r := recover(); r != nil {
				golog.Error(`Panic: `, r)
			}
		}()
		act(pack, wsConn)
	}
}
