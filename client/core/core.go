package core

import (
	"Spark/client/common"
	"Spark/client/config"
	"Spark/client/service/basic"
	"Spark/client/service/file"
	"Spark/client/service/process"
	"Spark/client/service/screenshot"
	"Spark/client/service/terminal"
	"Spark/modules"
	"Spark/utils"
	"encoding/hex"
	"errors"
	ws "github.com/gorilla/websocket"
	"github.com/kataras/golog"
	"net/http"
	"os"
	"strconv"
	"time"
)

// simplified type of map
type smap map[string]interface{}

var stop bool
var (
	errNoSecretHeader = errors.New(`can not find secret header`)
)

func Start() {
	for !stop {
		var err error
		if common.WSConn != nil {
			common.WSConn.Close()
		}
		common.WSConn, err = connectWS()
		if err != nil && !stop {
			golog.Error(err)
			<-time.After(3 * time.Second)
			continue
		}

		err = reportWS(common.WSConn)
		if err != nil && !stop {
			golog.Error(err)
			<-time.After(3 * time.Second)
			continue
		}

		go heartbeat(common.WSConn)

		err = handleWS(common.WSConn)
		if err != nil && !stop {
			golog.Error(err)
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
	return &common.Conn{Conn: wsConn, Secret: secret}, nil
}

func reportWS(wsConn *common.Conn) error {
	device, err := GetDevice()
	if err != nil {
		return err
	}
	pack := modules.CommonPack{Act: `report`, Data: device}
	err = common.SendPack(pack, wsConn)
	common.WSConn.SetWriteDeadline(time.Time{})
	if err != nil {
		return err
	}
	common.WSConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, data, err := common.WSConn.ReadMessage()
	common.WSConn.SetReadDeadline(time.Time{})
	if err != nil {
		return err
	}
	data, err = utils.Decrypt(data, common.WSConn.Secret)
	if err != nil {
		return err
	}
	err = utils.JSON.Unmarshal(data, &pack)
	if err != nil {
		return err
	}
	if pack.Code != 0 {
		return errors.New(`unknown error occurred`)
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
		data, err = utils.Decrypt(data, wsConn.Secret)
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
	switch pack.Act {
	case `offline`:
		common.SendCb(modules.Packet{Code: 0}, pack, wsConn)
		stop = true
		wsConn.Close()
		os.Exit(0)
		return
	case `lock`:
		err := basic.Lock()
		if err != nil {
			common.SendCb(modules.Packet{Code: 1, Msg: err.Error()}, pack, wsConn)
		} else {
			common.SendCb(modules.Packet{Code: 0}, pack, wsConn)
		}
	case `logoff`:
		err := basic.Logoff()
		if err != nil {
			common.SendCb(modules.Packet{Code: 1, Msg: err.Error()}, pack, wsConn)
		} else {
			common.SendCb(modules.Packet{Code: 0}, pack, wsConn)
		}
	case `hibernate`:
		err := basic.Hibernate()
		if err != nil {
			common.SendCb(modules.Packet{Code: 1, Msg: err.Error()}, pack, wsConn)
		} else {
			common.SendCb(modules.Packet{Code: 0}, pack, wsConn)
		}
	case `suspend`:
		err := basic.Suspend()
		if err != nil {
			common.SendCb(modules.Packet{Code: 1, Msg: err.Error()}, pack, wsConn)
		} else {
			common.SendCb(modules.Packet{Code: 0}, pack, wsConn)
		}
	case `restart`:
		err := basic.Restart()
		if err != nil {
			common.SendCb(modules.Packet{Code: 1, Msg: err.Error()}, pack, wsConn)
		} else {
			common.SendCb(modules.Packet{Code: 0}, pack, wsConn)
		}
	case `shutdown`:
		err := basic.Shutdown()
		if err != nil {
			common.SendCb(modules.Packet{Code: 1, Msg: err.Error()}, pack, wsConn)
		} else {
			common.SendCb(modules.Packet{Code: 0}, pack, wsConn)
		}
	case `screenshot`:
		if len(pack.Event) > 0 {
			screenshot.GetScreenshot(pack.Event)
		}
	case `initTerminal`:
		err := terminal.InitTerminal(pack)
		if err != nil {
			common.SendCb(modules.Packet{Act: `initTerminal`, Code: 1, Msg: err.Error()}, pack, wsConn)
		}
		break
	case `inputTerminal`:
		terminal.InputTerminal(pack)
		break
	case `killTerminal`:
		terminal.KillTerminal(pack)
		break
	case `listFiles`:
		path := `/`
		if val, ok := pack.Data[`path`]; ok {
			if path, ok = val.(string); !ok {
				path = `/`
			}
		}
		files, err := file.ListFiles(path)
		if err != nil {
			common.SendCb(modules.Packet{Code: 1, Msg: err.Error()}, pack, wsConn)
		} else {
			common.SendCb(modules.Packet{Code: 0, Data: smap{`files`: files}}, pack, wsConn)
		}
	case `removeFile`:
		path, ok := pack.Data[`path`]
		if !ok {
			common.SendCb(modules.Packet{Code: 1, Msg: `can not find such a file or directory`}, pack, wsConn)
			return
		}
		if path == `\` || path == `/` || len(path.(string)) == 0 {
			common.SendCb(modules.Packet{Code: 1, Msg: `can not find such a file or directory`}, pack, wsConn)
			return
		}
		err := os.RemoveAll(path.(string))
		if err != nil {
			common.SendCb(modules.Packet{Code: 1, Msg: err.Error()}, pack, wsConn)
		} else {
			common.SendCb(modules.Packet{Code: 0}, pack, wsConn)
		}
	case `uploadFile`:
		var start, end int64
		var path string
		{
			tempVal, ok := pack.Data[`file`]
			if !ok {
				common.SendCb(modules.Packet{Code: 1, Msg: `未知错误`}, pack, wsConn)
				return
			}
			if path, ok = tempVal.(string); !ok {
				common.SendCb(modules.Packet{Code: 1, Msg: `未知错误`}, pack, wsConn)
				return
			}
			tempVal, ok = pack.Data[`start`]
			if ok {
				if v, ok := tempVal.(float64); ok {
					start = int64(v)
				}
			}
			tempVal, ok = pack.Data[`end`]
			if ok {
				if v, ok := tempVal.(float64); ok {
					end = int64(v)
					if end > 0 {
						end++
					}
				}
			}
			if end > 0 && end < start {
				common.SendCb(modules.Packet{Code: 1, Msg: `文件范围错误`}, pack, wsConn)
				return
			}
		}
		err := file.UploadFile(path, pack.Event, start, end)
		if err != nil {
			common.SendCb(modules.Packet{Code: 1, Msg: err.Error()}, pack, wsConn)
		}
	case `listProcesses`:
		processes, err := process.ListProcesses()
		if err != nil {
			common.SendCb(modules.Packet{Code: 1, Msg: err.Error()}, pack, wsConn)
		} else {
			common.SendCb(modules.Packet{Code: 0, Data: map[string]interface{}{`processes`: processes}}, pack, wsConn)
		}
	case `killProcess`:
		pidStr, ok := pack.Data[`pid`]
		if !ok {
			common.SendCb(modules.Packet{Code: 1, Msg: `未知错误`}, pack, wsConn)
			return
		}
		pid, err := strconv.ParseInt(pidStr.(string), 10, 32)
		if err != nil {
			common.SendCb(modules.Packet{Code: 1, Msg: `未知错误`}, pack, wsConn)
			return
		}
		err = process.KillProcess(int32(pid))
		if err != nil {
			common.SendCb(modules.Packet{Code: 1, Msg: err.Error()}, pack, wsConn)
		} else {
			common.SendCb(modules.Packet{Code: 0}, pack, wsConn)
		}
	default:
		common.SendCb(modules.Packet{Code: 0}, pack, wsConn)
	}
	return
}

func heartbeat(wsConn *common.Conn) error {
	t := 0
	for range time.NewTicker(5 * time.Second).C {
		t++
		// Get disk info every 30*5 seconds.
		device, err := GetPartialInfo(t >= 30)
		if err != nil {
			golog.Error(err)
			continue
		}
		if t >= 30 {
			t = 0
		}
		err = common.SendPack(modules.CommonPack{Act: `setDevice`, Data: device}, wsConn)
		if err != nil {
			return err
		}
	}
	return nil
}
