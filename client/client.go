package main

import (
	"Spark/client/config"
	"Spark/client/core"
	"Spark/utils"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"math/big"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/kataras/golog"
)

func init() {
	golog.SetTimeFormat(`2006/01/02 15:04:05`)

	if len(strings.Trim(config.ConfigBuffer, "\x19")) == 0 {
		os.Exit(0)
		return
	}

	// Convert first 2 bytes to int, which is the length of the encrypted config.
	dataLen := int(big.NewInt(0).SetBytes([]byte(config.ConfigBuffer[:2])).Uint64())
	if dataLen > len(config.ConfigBuffer)-2 {
		os.Exit(1)
		return
	}
	cfgBytes := utils.StringToBytes(config.ConfigBuffer, 2, 2+dataLen)
	cfgBytes, err := decrypt(cfgBytes[16:], cfgBytes[:16])
	if err != nil {
		os.Exit(1)
		return
	}
	err = utils.JSON.Unmarshal(cfgBytes, &config.Config)
	if err != nil {
		os.Exit(1)
		return
	}
	if strings.HasSuffix(config.Config.Path, `/`) {
		config.Config.Path = config.Config.Path[:len(config.Config.Path)-1]
	}
}

func main() {
	update()
	core.Start()
}

func update() {
	selfPath, err := os.Executable()
	if err != nil {
		selfPath = os.Args[0]
	}
	if len(os.Args) > 1 && os.Args[1] == `--update` {
		if len(selfPath) <= 4 {
			return
		}
		destPath := selfPath[:len(selfPath)-4]
		thisFile, err := os.ReadFile(selfPath)
		if err != nil {
			return
		}
		os.WriteFile(destPath, thisFile, 0755)
		cmd := exec.Command(destPath, `--clean`)
		if cmd.Start() == nil {
			os.Exit(0)
			return
		}
	}
	if len(os.Args) > 1 && os.Args[1] == `--clean` {
		<-time.After(3 * time.Second)
		os.Remove(selfPath + `.tmp`)
	}
}

func decrypt(data []byte, key []byte) ([]byte, error) {
	// MD5[16 bytes] + Data[n bytes]
	dataLen := len(data)
	if dataLen <= 16 {
		return nil, utils.ErrEntityInvalid
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	stream := cipher.NewCTR(block, data[:16])
	decBuffer := make([]byte, dataLen-16)
	stream.XORKeyStream(decBuffer, data[16:])
	hash, _ := utils.GetMD5(decBuffer)
	if !bytes.Equal(hash, data[:16]) {
		return nil, utils.ErrFailedVerification
	}
	return decBuffer[:dataLen-16], nil
}
