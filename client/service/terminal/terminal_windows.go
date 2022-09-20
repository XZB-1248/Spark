package terminal

import (
	"Spark/client/common"
	"Spark/modules"
	"encoding/hex"
	"io"
	"os/exec"
	"reflect"
	"syscall"
	"time"
)

type terminal struct {
	lastPack int64
	event    string
	cmd      *exec.Cmd
	stdout   *io.ReadCloser
	stderr   *io.ReadCloser
	stdin    *io.WriteCloser
}

var defaultCmd = ``

func init() {
	defer func() {
		recover()
	}()
	syscall.NewLazyDLL(`kernel32.dll`).NewProc(`SetConsoleCP`).Call(65001)
	go healthCheck()
}

func InitTerminal(pack modules.Packet) error {
	cmd := exec.Command(getTerminal())
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cmd.Process.Kill()
		cmd.Process.Release()
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cmd.Process.Kill()
		cmd.Process.Release()
		return err
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cmd.Process.Kill()
		cmd.Process.Release()
		return err
	}
	termSession := &terminal{
		cmd:      cmd,
		event:    pack.Event,
		stdout:   &stdout,
		stderr:   &stderr,
		stdin:    &stdin,
		lastPack: common.Unix,
	}
	terminals.Set(pack.Data[`terminal`].(string), termSession)

	readSender := func(rc io.ReadCloser) {
		for {
			buffer := make([]byte, 512)
			n, err := rc.Read(buffer)
			buffer = buffer[:n]

			common.WSConn.SendCallback(modules.Packet{Act: `outputTerminal`, Data: map[string]any{
				`output`: hex.EncodeToString(buffer),
			}}, pack)
			termSession.lastPack = common.Unix
			if err != nil {
				common.WSConn.SendCallback(modules.Packet{Act: `quitTerminal`}, pack)
				break
			}
		}
	}
	go readSender(stdout)
	go readSender(stderr)

	cmd.Start()
	return nil
}

func InputTerminal(pack modules.Packet) error {
	val, ok := pack.GetData(`input`, reflect.String)
	if !ok {
		return errDataNotFound
	}
	data, err := hex.DecodeString(val.(string))
	if err != nil {
		return errDataInvalid
	}

	val, ok = pack.GetData(`terminal`, reflect.String)
	if !ok {
		return errUUIDNotFound
	}
	termUUID := val.(string)
	val, ok = terminals.Get(termUUID)
	if !ok {
		common.WSConn.SendCallback(modules.Packet{Act: `quitTerminal`, Msg: `${i18n|terminalSessionClosed}`}, pack)
		return nil
	}
	terminal := val.(*terminal)
	(*terminal.stdin).Write(data)
	terminal.lastPack = common.Unix
	return nil
}

func ResizeTerminal(pack modules.Packet) error {
	return nil
}

func KillTerminal(pack modules.Packet) error {
	val, ok := pack.GetData(`terminal`, reflect.String)
	if !ok {
		return errUUIDNotFound
	}
	termUUID := val.(string)
	val, ok = terminals.Get(termUUID)
	if !ok {
		common.WSConn.SendCallback(modules.Packet{Act: `quitTerminal`, Msg: `${i18n|terminalSessionClosed}`}, pack)
		return nil
	}
	terminal := val.(*terminal)
	terminals.Remove(termUUID)
	doKillTerminal(terminal)
	return nil
}

func PingTerminal(pack modules.Packet) {
	var termUUID string
	var termSession *terminal
	if val, ok := pack.GetData(`terminal`, reflect.String); !ok {
		return
	} else {
		termUUID = val.(string)
	}
	if val, ok := terminals.Get(termUUID); !ok {
		return
	} else {
		termSession = val.(*terminal)
		termSession.lastPack = common.Unix
	}
}

func doKillTerminal(terminal *terminal) {
	(*terminal.stdout).Close()
	(*terminal.stderr).Close()
	(*terminal.stdin).Close()
	if terminal.cmd.Process != nil {
		terminal.cmd.Process.Kill()
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
		terminals.IterCb(func(uuid string, t any) bool {
			termSession := t.(*terminal)
			if timestamp-termSession.lastPack > MaxInterval {
				keys = append(keys, uuid)
				doKillTerminal(termSession)
			}
			return true
		})
		terminals.Remove(keys...)
	}
}
