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
	"os/exec"
	"reflect"
	"strings"
)

var handlers = map[string]func(pack modules.Packet, wsConn *common.Conn){
	`ping`:           ping,
	`offline`:        offline,
	`lock`:           lock,
	`logoff`:         logoff,
	`hibernate`:      hibernate,
	`suspend`:        suspend,
	`restart`:        restart,
	`shutdown`:       shutdown,
	`screenshot`:     screenshot,
	`initTerminal`:   initTerminal,
	`inputTerminal`:  inputTerminal,
	`resizeTerminal`: resizeTerminal,
	`pingTerminal`:   pingTerminal,
	`killTerminal`:   killTerminal,
	`listFiles`:      listFiles,
	`fetchFile`:      fetchFile,
	`removeFiles`:    removeFiles,
	`uploadFiles`:    uploadFiles,
	`uploadTextFile`: uploadTextFile,
	`listProcesses`:  listProcesses,
	`killProcess`:    killProcess,
	`initDesktop`:    initDesktop,
	`pingDesktop`:    pingDesktop,
	`killDesktop`:    killDesktop,
	`getDesktop`:     getDesktop,
	`execCommand`:    execCommand,
}

func ping(pack modules.Packet, wsConn *common.Conn) {
	wsConn.SendCallback(modules.Packet{Code: 0}, pack)
	device, err := GetPartialInfo()
	if err != nil {
		golog.Error(err)
		return
	}
	wsConn.SendPack(modules.CommonPack{Act: `setDevice`, Data: *device})
}

func offline(pack modules.Packet, wsConn *common.Conn) {
	wsConn.SendCallback(modules.Packet{Code: 0}, pack)
	stop = true
	wsConn.Close()
	os.Exit(0)
}

func lock(pack modules.Packet, wsConn *common.Conn) {
	err := basic.Lock()
	if err != nil {
		wsConn.SendCallback(modules.Packet{Code: 1, Msg: err.Error()}, pack)
	} else {
		wsConn.SendCallback(modules.Packet{Code: 0}, pack)
	}
}

func logoff(pack modules.Packet, wsConn *common.Conn) {
	err := basic.Logoff()
	if err != nil {
		wsConn.SendCallback(modules.Packet{Code: 1, Msg: err.Error()}, pack)
	} else {
		wsConn.SendCallback(modules.Packet{Code: 0}, pack)
	}
}

func hibernate(pack modules.Packet, wsConn *common.Conn) {
	err := basic.Hibernate()
	if err != nil {
		wsConn.SendCallback(modules.Packet{Code: 1, Msg: err.Error()}, pack)
	} else {
		wsConn.SendCallback(modules.Packet{Code: 0}, pack)
	}
}

func suspend(pack modules.Packet, wsConn *common.Conn) {
	err := basic.Suspend()
	if err != nil {
		wsConn.SendCallback(modules.Packet{Code: 1, Msg: err.Error()}, pack)
	} else {
		wsConn.SendCallback(modules.Packet{Code: 0}, pack)
	}
}

func restart(pack modules.Packet, wsConn *common.Conn) {
	err := basic.Restart()
	if err != nil {
		wsConn.SendCallback(modules.Packet{Code: 1, Msg: err.Error()}, pack)
	} else {
		wsConn.SendCallback(modules.Packet{Code: 0}, pack)
	}
}

func shutdown(pack modules.Packet, wsConn *common.Conn) {
	err := basic.Shutdown()
	if err != nil {
		wsConn.SendCallback(modules.Packet{Code: 1, Msg: err.Error()}, pack)
	} else {
		wsConn.SendCallback(modules.Packet{Code: 0}, pack)
	}
}

func screenshot(pack modules.Packet, wsConn *common.Conn) {
	var bridge string
	if val, ok := pack.GetData(`bridge`, reflect.String); !ok {
		wsConn.SendCallback(modules.Packet{Code: 1, Msg: `${i18n|invalidParameter}`}, pack)
		return
	} else {
		bridge = val.(string)
	}
	err := Screenshot.GetScreenshot(bridge)
	if err != nil {
		wsConn.SendCallback(modules.Packet{Code: 1, Msg: err.Error()}, pack)
	}
}

func initTerminal(pack modules.Packet, wsConn *common.Conn) {
	err := terminal.InitTerminal(pack)
	if err != nil {
		wsConn.SendCallback(modules.Packet{Act: `initTerminal`, Code: 1, Msg: err.Error()}, pack)
	} else {
		wsConn.SendCallback(modules.Packet{Act: `initTerminal`, Code: 0}, pack)
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
		wsConn.SendCallback(modules.Packet{Code: 1, Msg: err.Error()}, pack)
	} else {
		wsConn.SendCallback(modules.Packet{Code: 0, Data: smap{`files`: files}}, pack)
	}
}

func fetchFile(pack modules.Packet, wsConn *common.Conn) {
	var path, filename, bridge string
	if val, ok := pack.GetData(`path`, reflect.String); !ok {
		wsConn.SendCallback(modules.Packet{Code: 1, Msg: `${i18n|fileOrDirNotExist}`}, pack)
		return
	} else {
		path = val.(string)
	}
	if val, ok := pack.GetData(`file`, reflect.String); !ok {
		wsConn.SendCallback(modules.Packet{Code: 1, Msg: `${i18n|invalidParameter}`}, pack)
		return
	} else {
		filename = val.(string)
	}
	if val, ok := pack.GetData(`bridge`, reflect.String); !ok {
		wsConn.SendCallback(modules.Packet{Code: 1, Msg: `${i18n|invalidParameter}`}, pack)
		return
	} else {
		bridge = val.(string)
	}
	err := file.FetchFile(path, filename, bridge)
	if err != nil {
		wsConn.SendCallback(modules.Packet{Code: 1, Msg: err.Error()}, pack)
	}
}

func removeFiles(pack modules.Packet, wsConn *common.Conn) {
	var files []string
	if val, ok := pack.Data[`files`]; !ok {
		wsConn.SendCallback(modules.Packet{Code: 1, Msg: `${i18n|fileOrDirNotExist}`}, pack)
		return
	} else {
		slice := val.([]any)
		for i := 0; i < len(slice); i++ {
			file, ok := slice[i].(string)
			if ok {
				files = append(files, file)
			}
		}
		if len(files) == 0 {
			wsConn.SendCallback(modules.Packet{Code: 1, Msg: `${i18n|fileOrDirNotExist}`}, pack)
			return
		}
	}
	err := file.RemoveFiles(files)
	if err != nil {
		wsConn.SendCallback(modules.Packet{Code: 1, Msg: err.Error()}, pack)
	} else {
		wsConn.SendCallback(modules.Packet{Code: 0}, pack)
	}
}

func uploadFiles(pack modules.Packet, wsConn *common.Conn) {
	var (
		start, end int64
		files      []string
		bridge     string
	)
	if val, ok := pack.Data[`files`]; !ok {
		wsConn.SendCallback(modules.Packet{Code: 1, Msg: `${i18n|fileOrDirNotExist}`}, pack)
		return
	} else {
		slice := val.([]any)
		for i := 0; i < len(slice); i++ {
			file, ok := slice[i].(string)
			if ok {
				files = append(files, file)
			}
		}
		if len(files) == 0 {
			wsConn.SendCallback(modules.Packet{Code: 1, Msg: `${i18n|fileOrDirNotExist}`}, pack)
			return
		}
	}
	if val, ok := pack.GetData(`bridge`, reflect.String); !ok {
		wsConn.SendCallback(modules.Packet{Code: 1, Msg: `${i18n|invalidParameter}`}, pack)
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
			wsConn.SendCallback(modules.Packet{Code: 1, Msg: `${i18n|invalidFileRange}`}, pack)
			return
		}
	}
	err := file.UploadFiles(files, bridge, start, end)
	if err != nil {
		golog.Error(err)
		wsConn.SendCallback(modules.Packet{Code: 1, Msg: err.Error()}, pack)
	}
}

func uploadTextFile(pack modules.Packet, wsConn *common.Conn) {
	var path, bridge string
	if val, ok := pack.GetData(`file`, reflect.String); !ok {
		wsConn.SendCallback(modules.Packet{Code: 1, Msg: `${i18n|fileOrDirNotExist}`}, pack)
		return
	} else {
		path = val.(string)
	}
	if val, ok := pack.GetData(`bridge`, reflect.String); !ok {
		wsConn.SendCallback(modules.Packet{Code: 1, Msg: `${i18n|invalidParameter}`}, pack)
		return
	} else {
		bridge = val.(string)
	}
	err := file.UploadTextFile(path, bridge)
	if err != nil {
		golog.Error(err)
		wsConn.SendCallback(modules.Packet{Code: 1, Msg: err.Error()}, pack)
	}
}

func listProcesses(pack modules.Packet, wsConn *common.Conn) {
	processes, err := process.ListProcesses()
	if err != nil {
		wsConn.SendCallback(modules.Packet{Code: 1, Msg: err.Error()}, pack)
	} else {
		wsConn.SendCallback(modules.Packet{Code: 0, Data: map[string]any{`processes`: processes}}, pack)
	}
}

func killProcess(pack modules.Packet, wsConn *common.Conn) {
	var (
		pid int32
		err error
	)
	if val, ok := pack.GetData(`pid`, reflect.Float64); !ok {
		wsConn.SendCallback(modules.Packet{Code: 1, Msg: `${i18n|invalidParameter}`}, pack)
		return
	} else {
		pid = int32(val.(float64))
	}
	err = process.KillProcess(int32(pid))
	if err != nil {
		wsConn.SendCallback(modules.Packet{Code: 1, Msg: err.Error()}, pack)
	} else {
		wsConn.SendCallback(modules.Packet{Code: 0}, pack)
	}
}

func initDesktop(pack modules.Packet, wsConn *common.Conn) {
	err := desktop.InitDesktop(pack)
	if err != nil {
		wsConn.SendCallback(modules.Packet{Act: `initDesktop`, Code: 1, Msg: err.Error()}, pack)
	} else {
		wsConn.SendCallback(modules.Packet{Act: `initDesktop`, Code: 0}, pack)
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

func execCommand(pack modules.Packet, wsConn *common.Conn) {
	var proc *exec.Cmd
	var cmd, args string
	if val, ok := pack.Data[`cmd`]; !ok {
		wsConn.SendCallback(modules.Packet{Code: 1, Msg: `${i18n|invalidParameter}`}, pack)
		return
	} else {
		cmd = val.(string)
	}
	if val, ok := pack.Data[`args`]; !ok {
		wsConn.SendCallback(modules.Packet{Code: 1, Msg: `${i18n|invalidParameter}`}, pack)
		return
	} else {
		args = val.(string)
	}
	if len(args) == 0 {
		proc = exec.Command(cmd)
	} else {
		proc = exec.Command(cmd, strings.Split(args, ` `)...)
	}
	err := proc.Start()
	if err != nil {
		wsConn.SendCallback(modules.Packet{Code: 1, Msg: err.Error()}, pack)
	} else {
		wsConn.SendCallback(modules.Packet{Code: 0, Data: map[string]any{
			`pid`: proc.Process.Pid,
		}}, pack)
		proc.Process.Release()
	}
}
