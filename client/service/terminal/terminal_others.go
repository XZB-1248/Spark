//go:build !windows

package terminal

import (
	"Spark/client/common"
	"Spark/modules"
	"Spark/utils"
	"Spark/utils/cmap"
	"encoding/hex"
	"github.com/creack/pty"
	"os"
	"os/exec"
	"reflect"
	"time"
)

type terminal struct {
	escape   bool
	lastPack int64
	rawEvent []byte
	event    string
	pty      *os.File
	cmd      *exec.Cmd
}

var terminals = cmap.New[*terminal]()
var defaultShell = ``

func init() {
	go healthCheck()
}

func InitTerminal(pack modules.Packet) error {
	// try to get shell
	// if shell is not found or unavailable, then fallback to `sh`
	cmd := exec.Command(getTerminal(false))
	ptySession, err := pty.Start(cmd)
	if err != nil {
		defaultShell = getTerminal(true)
		return err
	}
	rawEvent, _ := hex.DecodeString(pack.Event)
	session := &terminal{
		cmd:      cmd,
		pty:      ptySession,
		event:    pack.Event,
		lastPack: utils.Unix,
		rawEvent: rawEvent,
		escape:   false,
	}
	terminals.Set(pack.Data[`terminal`].(string), session)
	go func() {
		bufSize := 1024
		for !session.escape {
			buffer := make([]byte, bufSize)
			n, err := ptySession.Read(buffer)
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
				if !session.escape {
					session.escape = true
					doKillTerminal(session)
				}
				data, _ := utils.JSON.Marshal(modules.Packet{Act: `TERMINAL_QUIT`})
				data = utils.XOR(data, common.WSConn.GetSecret())
				common.WSConn.SendRawData(session.rawEvent, data, 21, 01)
				break
			}
		}
	}()

	return nil
}

func InputRawTerminal(input []byte, uuid string) {
	session, ok := terminals.Get(uuid)
	if !ok {
		return
	}
	session.pty.Write(input)
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
	session.pty.Write(input)
	session.lastPack = utils.Unix
}

func ResizeTerminal(pack modules.Packet) {
	var uuid string
	var cols, rows uint16
	var session *terminal
	if val, ok := pack.GetData(`cols`, reflect.Float64); !ok {
		return
	} else {
		cols = uint16(val.(float64))
	}
	if val, ok := pack.GetData(`rows`, reflect.Float64); !ok {
		return
	} else {
		rows = uint16(val.(float64))
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
	pty.Setsize(session.pty, &pty.Winsize{
		Cols: cols,
		Rows: rows,
	})
}

func KillTerminal(pack modules.Packet) {
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
	terminals.Remove(uuid)
	data, _ := utils.JSON.Marshal(modules.Packet{Act: `TERMINAL_QUIT`, Msg: `${i18n|TERMINAL.SESSION_CLOSED}`})
	data = utils.XOR(data, common.WSConn.GetSecret())
	common.WSConn.SendRawData(session.rawEvent, data, 21, 01)
	session.escape = true
	session.rawEvent = nil
}

func PingTerminal(pack modules.Packet) {
	var termUUID string
	if val, ok := pack.GetData(`terminal`, reflect.String); !ok {
		return
	} else {
		termUUID = val.(string)
	}
	session, ok := terminals.Get(termUUID)
	if !ok {
		return
	}
	session.lastPack = utils.Unix
}

func doKillTerminal(terminal *terminal) {
	terminal.escape = true
	if terminal.pty != nil {
		terminal.pty.Close()
	}
	if terminal.cmd.Process != nil {
		terminal.cmd.Process.Kill()
		terminal.cmd.Process.Wait()
		terminal.cmd.Process.Release()
		terminal.cmd.Process = nil
	}
}

func getTerminal(sh bool) string {
	shellTable := []string{`zsh`, `bash`, `sh`}
	if sh {
		shPath, err := exec.LookPath(`sh`)
		if err != nil {
			return `sh`
		}
		return shPath
	} else if len(defaultShell) > 0 {
		return defaultShell
	}
	for i := 0; i < len(shellTable); i++ {
		shellPath, err := exec.LookPath(shellTable[i])
		if err == nil {
			defaultShell = shellPath
			return shellPath
		}
	}
	return `sh`
}

func healthCheck() {
	const MaxInterval = 300
	for now := range time.NewTicker(30 * time.Second).C {
		timestamp := now.Unix()
		// stores sessions to be disconnected
		queue := make([]string, 0)
		terminals.IterCb(func(uuid string, session *terminal) bool {
			if timestamp-session.lastPack > MaxInterval {
				queue = append(queue, uuid)
				doKillTerminal(session)
			}
			return true
		})
		for i := 0; i < len(queue); i++ {
			terminals.Remove(queue[i])
		}
	}
}
