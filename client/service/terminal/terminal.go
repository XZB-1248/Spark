package terminal

import (
	"Spark/client/common"
	"Spark/modules"
	"Spark/utils/cmap"
	"bytes"
	"encoding/hex"
	"errors"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"time"
)

type terminal struct {
	lastInput int64
	event     string
	cmd       *exec.Cmd
	stdout    *io.ReadCloser
	stderr    *io.ReadCloser
	stdin     *io.WriteCloser
}

var terminals = cmap.New()
var (
	errDataNotFound = errors.New(`no input found in packet`)
	errDataInvalid  = errors.New(`can not parse data in packet`)
	errUUIDNotFound = errors.New(`can not find terminal identifier`)
)

func init() {
	go healthCheck()
}

func InitTerminal(pack modules.Packet) error {
	cmd := exec.Command(getTerminal())
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cmd.Process.Kill()
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cmd.Process.Kill()
		return err
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cmd.Process.Kill()
		return err
	}
	go func() {
		for {
			buffer := make([]byte, 512)
			n, err := stdout.Read(buffer)
			buffer = buffer[:n]
			buffer, _ = gbkToUtf8(buffer)
			common.SendCb(modules.Packet{Act: `outputTerminal`, Data: map[string]interface{}{
				`output`: hex.EncodeToString(buffer),
			}}, pack, common.WSConn)
			if err != nil {
				common.SendCb(modules.Packet{Act: `quitTerminal`}, pack, common.WSConn)
				break
			}
		}
	}()
	go func() {
		for {
			buffer := make([]byte, 512)
			n, err := stderr.Read(buffer)
			buffer = buffer[:n]
			buffer, _ = gbkToUtf8(buffer)
			common.SendCb(modules.Packet{Act: `outputTerminal`, Data: map[string]interface{}{
				`output`: hex.EncodeToString(buffer),
			}}, pack, common.WSConn)
			if err != nil {
				common.SendCb(modules.Packet{Act: `quitTerminal`}, pack, common.WSConn)
				break
			}
		}
	}()

	event := ``
	if pack.Data != nil {
		if val, ok := pack.Data[`event`]; ok {
			event, _ = val.(string)
		}
	}
	terminals.Set(pack.Data[`terminal`].(string), &terminal{
		cmd:       cmd,
		event:     event,
		stdout:    &stdout,
		stderr:    &stderr,
		stdin:     &stdin,
		lastInput: time.Now().Unix(),
	})
	cmd.Start()
	return nil
}

func InputTerminal(pack modules.Packet) error {
	if pack.Data == nil {
		return errDataNotFound
	}
	val, ok := pack.Data[`input`]
	if !ok {
		return errDataNotFound
	}
	hexStr, ok := val.(string)
	if !ok {
		return errDataNotFound
	}
	data, err := hex.DecodeString(hexStr)
	if err != nil {
		return errDataInvalid
	}

	val, ok = pack.Data[`terminal`]
	if !ok {
		return errUUIDNotFound
	}
	termUUID, ok := val.(string)
	if !ok {
		return errUUIDNotFound
	}
	val, ok = terminals.Get(termUUID)
	if !ok {
		common.SendCb(modules.Packet{Act: `quitTerminal`, Msg: `终端已退出`}, pack, common.WSConn)
		return nil
	}
	terminal, ok := val.(*terminal)
	if !ok {
		common.SendCb(modules.Packet{Act: `quitTerminal`, Msg: `终端已退出`}, pack, common.WSConn)
		return nil
	}

	terminal.lastInput = time.Now().Unix()
	if len(data) == 1 && data[0] == '\x03' {
		terminal.cmd.Process.Signal(os.Interrupt)
		return nil
	}
	data, _ = utf8ToGbk(data)
	(*terminal.stdin).Write(data)
	return nil
}

func KillTerminal(pack modules.Packet) error {
	if pack.Data == nil {
		return errUUIDNotFound
	}
	val, ok := pack.Data[`terminal`]
	if !ok {
		return errUUIDNotFound
	}
	termUUID, ok := val.(string)
	if !ok {
		return errUUIDNotFound
	}
	val, ok = terminals.Get(termUUID)
	if !ok {
		common.SendCb(modules.Packet{Act: `quitTerminal`, Msg: `终端已退出`}, pack, common.WSConn)
		return nil
	}
	terminal, ok := val.(*terminal)
	if !ok {
		terminals.Remove(termUUID)
		common.SendCb(modules.Packet{Act: `quitTerminal`, Msg: `终端已退出`}, pack, common.WSConn)
		return nil
	}
	doKillTerminal(terminal)
	return nil
}

func doKillTerminal(terminal *terminal) {
	(*terminal.stdout).Close()
	(*terminal.stderr).Close()
	(*terminal.stdin).Close()
	if terminal.cmd.Process != nil {
		terminal.cmd.Process.Kill()
	}
}

func getTerminal() string {
	switch runtime.GOOS {
	case `windows`:
		return `cmd.exe`
	case `linux`:
		return `sh`
	case `darwin`:
		return `sh`
	default:
		return `sh`
	}
}

func gbkToUtf8(s []byte) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader(s), simplifiedchinese.GB18030.NewDecoder())
	d, e := ioutil.ReadAll(reader)
	if e != nil {
		return nil, e
	}
	return d, nil
}

func utf8ToGbk(s []byte) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader(s), simplifiedchinese.GB18030.NewEncoder())
	d, e := ioutil.ReadAll(reader)
	if e != nil {
		return nil, e
	}
	return d, nil
}

func healthCheck() {
	const MaxInterval = 180
	for now := range time.NewTicker(30 * time.Second).C {
		timestamp := now.Unix()
		// stores sessions to be disconnected
		queue := make([]string, 0)
		terminals.IterCb(func(uuid string, t interface{}) bool {
			terminal, ok := t.(*terminal)
			if !ok {
				queue = append(queue, uuid)
				return true
			}
			if timestamp-terminal.lastInput > MaxInterval {
				queue = append(queue, uuid)
				doKillTerminal(terminal)
			}
			return true
		})
		for i := 0; i < len(queue); i++ {
			terminals.Remove(queue[i])
		}
	}
}
