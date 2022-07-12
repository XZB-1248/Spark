package core

import (
	"Spark/client/common"
	"Spark/client/service/basic"
	"Spark/client/service/desktop"
	"Spark/client/service/file"
	"Spark/client/service/process"
	Screenshot "Spark/client/service/screenshot"
	"Spark/client/service/terminal"
	"Spark/modules"
	"github.com/kataras/golog"
	"os"
	"reflect"
	"strconv"
)

func ping(pack modules.Packet, wsConn *common.Conn) {
	common.SendCb(modules.Packet{Code: 0}, pack, wsConn)
	device, err := GetPartialInfo()
	if err != nil {
		golog.Error(err)
		return
	}
	common.SendPack(modules.CommonPack{Act: `setDevice`, Data: *device}, wsConn)
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
	var bridge string
	if val, ok := pack.GetData(`bridge`, reflect.String); !ok {
		common.SendCb(modules.Packet{Code: 1, Msg: `${i18n|invalidParameter}`}, pack, wsConn)
		return
	} else {
		bridge = val.(string)
	}
	err := Screenshot.GetScreenshot(bridge)
	if err != nil {
		common.SendCb(modules.Packet{Code: 1, Msg: err.Error()}, pack, wsConn)
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

func pingTerminal(pack modules.Packet, wsConn *common.Conn) {
	terminal.PingTerminal(pack)
}

func killTerminal(pack modules.Packet, wsConn *common.Conn) {
	terminal.KillTerminal(pack)
}

func listFiles(pack modules.Packet, wsConn *common.Conn) {
	path := `/`
	if val, ok := pack.GetData(`path`, reflect.String); ok {
		path = val.(string)
	}
	files, err := file.ListFiles(path)
	if err != nil {
		common.SendCb(modules.Packet{Code: 1, Msg: err.Error()}, pack, wsConn)
	} else {
		common.SendCb(modules.Packet{Code: 0, Data: smap{`files`: files}}, pack, wsConn)
	}
}

func fetchFile(pack modules.Packet, wsConn *common.Conn) {
	var path, filename, bridge string
	if val, ok := pack.GetData(`path`, reflect.String); !ok {
		common.SendCb(modules.Packet{Code: 1, Msg: `${i18n|fileOrDirNotExist}`}, pack, wsConn)
		return
	} else {
		path = val.(string)
	}
	if val, ok := pack.GetData(`file`, reflect.String); !ok {
		common.SendCb(modules.Packet{Code: 1, Msg: `${i18n|invalidParameter}`}, pack, wsConn)
		return
	} else {
		filename = val.(string)
	}
	if val, ok := pack.GetData(`bridge`, reflect.String); !ok {
		common.SendCb(modules.Packet{Code: 1, Msg: `${i18n|invalidParameter}`}, pack, wsConn)
		return
	} else {
		bridge = val.(string)
	}
	err := file.FetchFile(path, filename, bridge)
	if err != nil {
		common.SendCb(modules.Packet{Code: 1, Msg: err.Error()}, pack, wsConn)
	}
}

func removeFiles(pack modules.Packet, wsConn *common.Conn) {
	var files []string
	if val, ok := pack.Data[`files`]; !ok {
		common.SendCb(modules.Packet{Code: 1, Msg: `${i18n|fileOrDirNotExist}`}, pack, wsConn)
		return
	} else {
		slice := val.([]interface{})
		for i := 0; i < len(slice); i++ {
			file, ok := slice[i].(string)
			if ok {
				files = append(files, file)
			}
		}
		if len(files) == 0 {
			common.SendCb(modules.Packet{Code: 1, Msg: `${i18n|fileOrDirNotExist}`}, pack, wsConn)
			return
		}
	}
	err := file.RemoveFiles(files)
	if err != nil {
		common.SendCb(modules.Packet{Code: 1, Msg: err.Error()}, pack, wsConn)
	} else {
		common.SendCb(modules.Packet{Code: 0}, pack, wsConn)
	}
}

func uploadFiles(pack modules.Packet, wsConn *common.Conn) {
	var start, end int64
	var files []string
	var bridge string
	if val, ok := pack.Data[`files`]; !ok {
		common.SendCb(modules.Packet{Code: 1, Msg: `${i18n|fileOrDirNotExist}`}, pack, wsConn)
		return
	} else {
		slice := val.([]interface{})
		for i := 0; i < len(slice); i++ {
			file, ok := slice[i].(string)
			if ok {
				files = append(files, file)
			}
		}
		if len(files) == 0 {
			common.SendCb(modules.Packet{Code: 1, Msg: `${i18n|fileOrDirNotExist}`}, pack, wsConn)
			return
		}
	}
	if val, ok := pack.GetData(`bridge`, reflect.String); !ok {
		common.SendCb(modules.Packet{Code: 1, Msg: `${i18n|invalidParameter}`}, pack, wsConn)
		return
	} else {
		bridge = val.(string)
	}
	{
		if val, ok := pack.GetData(`start`, reflect.Float64); ok {
			start = int64(val.(float64))
		}
		if val, ok := pack.GetData(`end`, reflect.Float64); ok {
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
	err := file.UploadFiles(files, bridge, start, end)
	if err != nil {
		golog.Error(err)
		common.SendCb(modules.Packet{Code: 1, Msg: err.Error()}, pack, wsConn)
	}
}

func uploadTextFile(pack modules.Packet, wsConn *common.Conn) {
	var path string
	var bridge string
	if val, ok := pack.GetData(`file`, reflect.String); !ok {
		common.SendCb(modules.Packet{Code: 1, Msg: `${i18n|fileOrDirNotExist}`}, pack, wsConn)
		return
	} else {
		path = val.(string)
	}
	if val, ok := pack.GetData(`bridge`, reflect.String); !ok {
		common.SendCb(modules.Packet{Code: 1, Msg: `${i18n|invalidParameter}`}, pack, wsConn)
		return
	} else {
		bridge = val.(string)
	}
	err := file.UploadTextFile(path, bridge)
	if err != nil {
		golog.Error(err)
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
	if val, ok := pack.GetData(`pid`, reflect.String); ok {
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

func initDesktop(pack modules.Packet, wsConn *common.Conn) {
	err := desktop.InitDesktop(pack)
	if err != nil {
		common.SendCb(modules.Packet{Act: `initDesktop`, Code: 1, Msg: err.Error()}, pack, wsConn)
	}
}

func pingDesktop(pack modules.Packet, wsConn *common.Conn) {
	desktop.PingDesktop(pack)
}

func killDesktop(pack modules.Packet, wsConn *common.Conn) {
	desktop.KillDesktop(pack)
}

func getDesktop(pack modules.Packet, wsConn *common.Conn) {
	desktop.GetDesktop(pack)
}
