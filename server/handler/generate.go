package handler

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
	errTooLargeEntity = errors.New(`length of data can not excess buffer size`)
)

//func init() {
//	clientUUID := utils.GetUUID()
//	clientKey, _ := common.EncAES(clientUUID, append([]byte("XZB_Spark"), bytes.Repeat([]byte{25}, 24-9)...))
//	cfg, _ := genConfig(clientCfg{
//		Secure: false,
//		Host:   "47.102.136.182",
//		Port:   1025,
//		Path:   "/",
//		UUID:   hex.EncodeToString(clientUUID),
//		Key:    hex.EncodeToString(clientKey),
//	})
//	output := ``
//	temp := hex.EncodeToString(cfg)
//	for i := 0; i < len(temp); i += 2 {
//		output += `\x` + temp[i:i+2]
//	}
//	ioutil.WriteFile(`./Client.cfg`, []byte(output), 0755)
//}

func checkClient(ctx *gin.Context) {
	var form struct {
		OS     string `json:"os" yaml:"os" form:"os" binding:"required"`
		Arch   string `json:"arch" yaml:"arch" form:"arch" binding:"required"`
		Host   string `json:"host" yaml:"host" form:"host" binding:"required"`
		Port   uint16 `json:"port" yaml:"port" form:"port" binding:"required"`
		Path   string `json:"path" yaml:"path" form:"path" binding:"required"`
		Secure string `json:"secure" yaml:"secure" form:"secure"`
	}
	if err := ctx.ShouldBind(&form); err != nil {
		ctx.JSON(http.StatusBadRequest, modules.Packet{Code: -1, Msg: `参数不完整`})
		return
	}
	_, err := common.BuiltFS.Open(fmt.Sprintf(`/%v_%v`, form.OS, form.Arch))
	if err != nil {
		ctx.JSON(http.StatusNotFound, modules.Packet{Code: 1, Msg: `该系统或架构的客户端尚未编译`})
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
		if err == errTooLargeEntity {
			ctx.JSON(http.StatusRequestEntityTooLarge, modules.Packet{Code: 1, Msg: `配置信息过长`})
			return
		}
		ctx.JSON(http.StatusInternalServerError, modules.Packet{Code: 1, Msg: `配置文件生成失败`})
		return
	}
	ctx.JSON(http.StatusOK, modules.Packet{Code: 0})
}

func generateClient(ctx *gin.Context) {
	var form struct {
		OS     string `json:"os" yaml:"os" form:"os" binding:"required"`
		Arch   string `json:"arch" yaml:"arch" form:"arch" binding:"required"`
		Host   string `json:"host" yaml:"host" form:"host" binding:"required"`
		Port   uint16 `json:"port" yaml:"port" form:"port" binding:"required"`
		Path   string `json:"path" yaml:"path" form:"path" binding:"required"`
		Secure string `json:"secure" yaml:"secure" form:"secure"`
	}
	if err := ctx.ShouldBind(&form); err != nil {
		ctx.JSON(http.StatusBadRequest, modules.Packet{Code: -1, Msg: `参数不完整`})
		return
	}
	tpl, err := common.BuiltFS.Open(fmt.Sprintf(`/%v_%v`, form.OS, form.Arch))
	if err != nil {
		ctx.JSON(http.StatusNotFound, modules.Packet{Code: 1, Msg: `该系统或架构的客户端尚未编译`})
		return
	}
	clientUUID := utils.GetUUID()
	clientKey, err := common.EncAES(clientUUID, config.Config.StdSalt)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, modules.Packet{Code: 1, Msg: `配置文件生成失败`})
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
		if err == errTooLargeEntity {
			ctx.JSON(http.StatusRequestEntityTooLarge, modules.Packet{Code: 1, Msg: `配置信息过长`})
			return
		}
		ctx.JSON(http.StatusInternalServerError, modules.Packet{Code: 1, Msg: `配置文件生成失败`})
		return
	}
	ctx.Header(`Accept-Ranges`, `none`)
	ctx.Header(`Content-Transfer-Encoding`, `binary`)
	ctx.Header(`Content-Type`, `application/octet-stream`)
	if stat, err := tpl.Stat(); err == nil {
		ctx.Header(`Content-Length`, strconv.FormatInt(stat.Size(), 10))
	}
	if form.OS == `windows` {
		ctx.Header(`Content-Disposition`, `attachment; filename=client.exe;`)
	} else {
		ctx.Header(`Content-Disposition`, `attachment; filename=client;`)
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
		return nil, errTooLargeEntity
	}

	dataLen := big.NewInt(int64(len(final))).Bytes()
	dataLen = append(bytes.Repeat([]byte{'\x00'}, 2-len(dataLen)), dataLen...)

	final = append(dataLen, final...)
	for len(final) < 384 {
		final = append(final, utils.GetUUID()...)
	}
	return final[:384], nil
}
