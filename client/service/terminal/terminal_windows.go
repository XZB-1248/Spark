package terminal

import (
	"Spark/client/common"
	"Spark/modules"
	"bytes"
	"encoding/hex"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"time"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

type terminal struct {
	lastPack int64
	event    string
	cmd      *exec.Cmd
	stdout   *io.ReadCloser
	stderr   *io.ReadCloser
	stdin    *io.WriteCloser
}

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
	termSession := &terminal{
		cmd:      cmd,
		event:    pack.Event,
		stdout:   &stdout,
		stderr:   &stderr,
		stdin:    &stdin,
		lastPack: time.Now().Unix(),
	}
	terminals.Set(pack.Data[`terminal`].(string), termSession)

	readSender := func(rc io.ReadCloser) {
		for {
			buffer := make([]byte, 512)
			n, err := rc.Read(buffer)
			buffer = buffer[:n]

			// Clear screen.
			if len(buffer) == 1 && buffer[0] == 12 {
				buffer = []byte{27, 91, 72, 27, 91, 50, 74}
			}

			buffer, _ = encodeUTF8(buffer)
			common.SendCb(modules.Packet{Act: `outputTerminal`, Data: map[string]interface{}{
				`output`: hex.EncodeToString(buffer),
			}}, pack, common.WSConn)
			termSession.lastPack = time.Now().Unix()
			if err != nil {
				common.SendCb(modules.Packet{Act: `quitTerminal`}, pack, common.WSConn)
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
		common.SendCb(modules.Packet{Act: `quitTerminal`, Msg: `${i18n|terminalSessionClosed}`}, pack, common.WSConn)
		return nil
	}
	terminal := val.(*terminal)
	terminal.lastPack = time.Now().Unix()
	if len(data) == 1 && data[0] == '\x03' {
		terminal.cmd.Process.Signal(os.Interrupt)
		return nil
	}
	data, _ = decodeUTF8(data)
	(*terminal.stdin).Write(data)
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
		common.SendCb(modules.Packet{Act: `quitTerminal`, Msg: `${i18n|terminalSessionClosed}`}, pack, common.WSConn)
		return nil
	}
	terminal := val.(*terminal)
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
	return `cmd.exe`
}

func encodeUTF8(s []byte) ([]byte, error) {
	if runtime.GOOS == `windows` {
		return gbkToUtf8(s)
	} else {
		return s, nil
	}
}

func decodeUTF8(s []byte) ([]byte, error) {
	if runtime.GOOS == `windows` {
		return utf8ToGbk(s)
	} else {
		return s, nil
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
	const MaxInterval = 300
	for now := range time.NewTicker(30 * time.Second).C {
		timestamp := now.Unix()
		// stores sessions to be disconnected
		queue := make([]string, 0)
		terminals.IterCb(func(uuid string, t interface{}) bool {
			termSession := t.(*terminal)
			if timestamp-termSession.lastPack > MaxInterval {
				queue = append(queue, uuid)
				doKillTerminal(termSession)
			}
			return true
		})
		for i := 0; i < len(queue); i++ {
			terminals.Remove(queue[i])
		}
	}
}
