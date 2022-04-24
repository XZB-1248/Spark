package core

import (
	"Spark/client/common"
	"Spark/client/service/basic"
	"Spark/client/service/file"
	"Spark/client/service/process"
	Screenshot "Spark/client/service/screenshot"
	"Spark/client/service/terminal"
	"Spark/modules"
	"os"
	"reflect"
	"strconv"
)

func getPackData(pack modules.Packet, key string, t reflect.Kind) (interface{}, bool) {
	data, ok := pack.Data[key]
	if !ok {
		return nil, false
	}
	switch t {
	case reflect.String:
		val, ok := data.(string)
		return val, ok
	case reflect.Uint:
		val, ok := data.(uint)
		return val, ok
	case reflect.Uint32:
		val, ok := data.(uint32)
		return val, ok
	case reflect.Uint64:
		val, ok := data.(uint64)
		return val, ok
	case reflect.Int:
		val, ok := data.(int)
		return val, ok
	case reflect.Int64:
		val, ok := data.(int64)
		return val, ok
	case reflect.Bool:
		val, ok := data.(bool)
		return val, ok
	case reflect.Float64:
		val, ok := data.(float64)
		return val, ok
	default:
		return nil, false
	}
}

func offline(pack modules.Packet, wsConn *common.Conn) {
	common.SendCb(modules.Packet{Code: 0}, pack, wsConn)
	stop = true
	wsConn.Close()
	os.Exit(0)
}

func lock(pack modules.Packet, wsConn *common.Conn) {
	err := basic.Lock()
	if err != nil {
		common.SendCb(modules.Packet{Code: 1, Msg: err.Error()}, pack, wsConn)
	} else {
		common.SendCb(modules.Packet{Code: 0}, pack, wsConn)
	}
}

func logoff(pack modules.Packet, wsConn *common.Conn) {
	err := basic.Logoff()
	if err != nil {
		common.SendCb(modules.Packet{Code: 1, Msg: err.Error()}, pack, wsConn)
	} else {
		common.SendCb(modules.Packet{Code: 0}, pack, wsConn)
	}
}

func hibernate(pack modules.Packet, wsConn *common.Conn) {
	err := basic.Hibernate()
	if err != nil {
		common.SendCb(modules.Packet{Code: 1, Msg: err.Error()}, pack, wsConn)
	} else {
		common.SendCb(modules.Packet{Code: 0}, pack, wsConn)
	}
}

func suspend(pack modules.Packet, wsConn *common.Conn) {
	err := basic.Suspend()
	if err != nil {
		common.SendCb(modules.Packet{Code: 1, Msg: err.Error()}, pack, wsConn)
	} else {
		common.SendCb(modules.Packet{Code: 0}, pack, wsConn)
	}
}

func restart(pack modules.Packet, wsConn *common.Conn) {
	err := basic.Restart()
	if err != nil {
		common.SendCb(modules.Packet{Code: 1, Msg: err.Error()}, pack, wsConn)
	} else {
		common.SendCb(modules.Packet{Code: 0}, pack, wsConn)
	}
}

func shutdown(pack modules.Packet, wsConn *common.Conn) {
	err := basic.Shutdown()
	if err != nil {
		common.SendCb(modules.Packet{Code: 1, Msg: err.Error()}, pack, wsConn)
	} else {
		common.SendCb(modules.Packet{Code: 0}, pack, wsConn)
	}
}

func screenshot(pack modules.Packet, wsConn *common.Conn) {
	if len(pack.Event) > 0 {
		Screenshot.GetScreenshot(pack.Event)
	}
}

func initTerminal(pack modules.Packet, wsConn *common.Conn) {
	err := terminal.InitTerminal(pack)
	if err != nil {
		common.SendCb(modules.Packet{Act: `initTerminal`, Code: 1, Msg: err.Error()}, pack, wsConn)
	}
}

func inputTerminal(pack modules.Packet, wsConn *common.Conn) {
	terminal.InputTerminal(pack)
}

func resizeTerminal(pack modules.Packet, wsConn *common.Conn) {
	terminal.ResizeTerminal(pack)
}

func killTerminal(pack modules.Packet, wsConn *common.Conn) {
	terminal.KillTerminal(pack)
}

func listFiles(pack modules.Packet, wsConn *common.Conn) {
	path := `/`
	if val, ok := getPackData(pack, `path`, reflect.String); ok {
		path = val.(string)
	}
	files, err := file.ListFiles(path)
	if err != nil {
		common.SendCb(modules.Packet{Code: 1, Msg: err.Error()}, pack, wsConn)
	} else {
		common.SendCb(modules.Packet{Code: 0, Data: smap{`files`: files}}, pack, wsConn)
	}
}

func removeFile(pack modules.Packet, wsConn *common.Conn) {
	var path string
	if val, ok := getPackData(pack, `file`, reflect.String); !ok {
		common.SendCb(modules.Packet{Code: 1, Msg: `${i18n|fileOrDirNotExist}`}, pack, wsConn)
		return
	} else {
		path = val.(string)
	}
	err := file.RemoveFile(path)
	if err != nil {
		common.SendCb(modules.Packet{Code: 1, Msg: err.Error()}, pack, wsConn)
	} else {
		common.SendCb(modules.Packet{Code: 0}, pack, wsConn)
	}
}

func uploadFile(pack modules.Packet, wsConn *common.Conn) {
	var start, end int64
	var path string
	if val, ok := getPackData(pack, `file`, reflect.String); !ok {
		common.SendCb(modules.Packet{Code: 1, Msg: `${i18n|fileOrDirNotExist}`}, pack, wsConn)
		return
	} else {
		path = val.(string)
	}
	{
		if val, ok := getPackData(pack, `start`, reflect.Float64); ok {
			start = int64(val.(float64))
		}
		if val, ok := getPackData(pack, `end`, reflect.Float64); ok {
			end = int64(val.(float64))
			if end > 0 {
				end++
			}
		}
		if end > 0 && end < start {
			common.SendCb(modules.Packet{Code: 1, Msg: `${i18n|invalidFileRange}`}, pack, wsConn)
			return
		}
	}
	err := file.UploadFile(path, pack.Event, start, end)
	if err != nil {
		common.SendCb(modules.Packet{Code: 1, Msg: err.Error()}, pack, wsConn)
	}
}

func listProcesses(pack modules.Packet, wsConn *common.Conn) {
	processes, err := process.ListProcesses()
	if err != nil {
		common.SendCb(modules.Packet{Code: 1, Msg: err.Error()}, pack, wsConn)
	} else {
		common.SendCb(modules.Packet{Code: 0, Data: map[string]interface{}{`processes`: processes}}, pack, wsConn)
	}
}

func killProcess(pack modules.Packet, wsConn *common.Conn) {
	var (
		pid int64
		err error
	)
	if val, ok := getPackData(pack, `pid`, reflect.String); ok {
		pid, err = strconv.ParseInt(val.(string), 10, 32)
		common.SendCb(modules.Packet{Code: 1, Msg: err.Error()}, pack, wsConn)
		return
	}
	err = process.KillProcess(int32(pid))
	if err != nil {
		common.SendCb(modules.Packet{Code: 1, Msg: err.Error()}, pack, wsConn)
	} else {
		common.SendCb(modules.Packet{Code: 0}, pack, wsConn)
	}
}
