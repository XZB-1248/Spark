package generate

import (
	"Spark/modules"
	"Spark/server/common"
	"Spark/server/config"
	"Spark/utils"
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type clientCfg struct {
	Secure bool   `json:"secure"`
	Host   string `json:"host"`
	Port   int    `json:"port"`
	Path   string `json:"path"`
	UUID   string `json:"uuid"`
	Key    string `json:"key"`
}

var (
	ErrTooLargeEntity = errors.New(`length of data can not excess buffer size`)
)

func CheckClient(ctx *gin.Context) {
	var form struct {
		OS     string `json:"os" yaml:"os" form:"os" binding:"required"`
		Arch   string `json:"arch" yaml:"arch" form:"arch" binding:"required"`
		Host   string `json:"host" yaml:"host" form:"host" binding:"required"`
		Port   uint16 `json:"port" yaml:"port" form:"port" binding:"required"`
		Path   string `json:"path" yaml:"path" form:"path" binding:"required"`
		Secure string `json:"secure" yaml:"secure" form:"secure"`
	}
	if err := ctx.ShouldBind(&form); err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, modules.Packet{Code: -1, Msg: `${i18n|invalidParameter}`})
		return
	}
	_, err := os.Stat(fmt.Sprintf(config.BuiltPath, form.OS, form.Arch))
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusNotFound, modules.Packet{Code: 1, Msg: `${i18n|osOrArchNotPrebuilt}`})
		return
	}
	_, err = genConfig(clientCfg{
		Secure: form.Secure == `true`,
		Host:   form.Host,
		Port:   int(form.Port),
		Path:   form.Path,
		UUID:   strings.Repeat(`FF`, 16),
		Key:    strings.Repeat(`FF`, 32),
	})
	if err != nil {
		if err == ErrTooLargeEntity {
			ctx.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, modules.Packet{Code: 1, Msg: `${i18n|tooLargeConfig}`})
			return
		}
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, modules.Packet{Code: 1, Msg: `${i18n|configGenerateFailed}`})
		return
	}
	ctx.JSON(http.StatusOK, modules.Packet{Code: 0})
}

func GenerateClient(ctx *gin.Context) {
	var form struct {
		OS     string `json:"os" yaml:"os" form:"os" binding:"required"`
		Arch   string `json:"arch" yaml:"arch" form:"arch" binding:"required"`
		Host   string `json:"host" yaml:"host" form:"host" binding:"required"`
		Port   uint16 `json:"port" yaml:"port" form:"port" binding:"required"`
		Path   string `json:"path" yaml:"path" form:"path" binding:"required"`
		Secure string `json:"secure" yaml:"secure" form:"secure"`
	}
	if err := ctx.ShouldBind(&form); err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, modules.Packet{Code: -1, Msg: `${i18n|invalidParameter}`})
		return
	}
	tpl, err := os.Open(fmt.Sprintf(config.BuiltPath, form.OS, form.Arch))
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusNotFound, modules.Packet{Code: 1, Msg: `${i18n|osOrArchNotPrebuilt}`})
		return
	}
	clientUUID := utils.GetUUID()
	clientKey, err := common.EncAES(clientUUID, config.Config.SaltBytes)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, modules.Packet{Code: 1, Msg: `${i18n|configGenerateFailed}`})
		return
	}
	cfgBytes, err := genConfig(clientCfg{
		Secure: form.Secure == `true`,
		Host:   form.Host,
		Port:   int(form.Port),
		Path:   form.Path,
		UUID:   hex.EncodeToString(clientUUID),
		Key:    hex.EncodeToString(clientKey),
	})
	if err != nil {
		if err == ErrTooLargeEntity {
			ctx.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, modules.Packet{Code: 1, Msg: `${i18n|tooLargeConfig}`})
			return
		}
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, modules.Packet{Code: 1, Msg: `${i18n|configGenerateFailed}`})
		return
	}
	ctx.Header(`Accept-Ranges`, `none`)
	ctx.Header(`Content-Transfer-Encoding`, `binary`)
	ctx.Header(`Content-Type`, `application/octet-stream`)
	if stat, err := tpl.Stat(); err == nil {
		ctx.Header(`Content-Length`, strconv.FormatInt(stat.Size(), 10))
	}
	if form.OS == `windows` {
		ctx.Header(`Content-Disposition`, `attachment; filename=client.exe; filename*=UTF-8''client.exe`)
	} else {
		ctx.Header(`Content-Disposition`, `attachment; filename=client; filename*=UTF-8''client`)
	}
	// Find and replace plain buffer with encrypted configuration.
	cfgBuffer := bytes.Repeat([]byte{'\x19'}, 384)
	prevBuffer := make([]byte, 0)
	for {
		thisBuffer := make([]byte, 1024)
		n, err := tpl.Read(thisBuffer)
		thisBuffer = thisBuffer[:n]
		tempBuffer := append(prevBuffer, thisBuffer...)
		bufIndex := bytes.Index(tempBuffer, cfgBuffer)
		if bufIndex > -1 {
			tempBuffer = bytes.Replace(tempBuffer, cfgBuffer, cfgBytes, -1)
		}
		ctx.Writer.Write(tempBuffer[:len(prevBuffer)])
		prevBuffer = tempBuffer[len(prevBuffer):]
		if err != nil {
			break
		}
	}
	if len(prevBuffer) > 0 {
		ctx.Writer.Write(prevBuffer)
		prevBuffer = []byte{}
	}
}

func genConfig(cfg clientCfg) ([]byte, error) {
	data, err := utils.JSON.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	key := utils.GetUUID()
	data, err = common.EncAES(data, key)
	if err != nil {
		return nil, err
	}
	final := append(key, data...)
	if len(final) > 384-2 {
		return nil, ErrTooLargeEntity
	}

	// Get the length of encrypted buffer as a 2-byte big-endian integer.
	// And append encrypted buffer to the end of the data length.
	dataLen := big.NewInt(int64(len(final))).Bytes()
	dataLen = append(bytes.Repeat([]byte{'\x00'}, 2-len(dataLen)), dataLen...)

	// If the length of encrypted buffer is less than 384,
	// append the remaining bytes with random bytes.
	final = append(dataLen, final...)
	for len(final) < 384 {
		final = append(final, utils.GetUUID()...)
	}
	return final[:384], nil
}
