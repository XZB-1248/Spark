package common

import (
	"Spark/modules"
	"Spark/utils"
	"Spark/utils/cmap"
	"Spark/utils/melody"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"github.com/gin-gonic/gin"
	"net"
	"reflect"
	"strings"
	"unsafe"
)

var Melody = melody.New()
var Devices = cmap.New()

func SendPackByUUID(pack modules.Packet, uuid string) bool {
	session, ok := Melody.GetSessionByUUID(uuid)
	if !ok {
		return false
	}
	return SendPack(pack, session)
}

func SendPack(pack modules.Packet, session *melody.Session) bool {
	if session == nil {
		return false
	}
	data, err := utils.JSON.Marshal(pack)
	if err != nil {
		return false
	}
	data, ok := Encrypt(data, session)
	if !ok {
		return false
	}
	err = session.WriteBinary(data)
	return err == nil
}

func Encrypt(data []byte, session *melody.Session) ([]byte, bool) {
	temp, ok := session.Get(`Secret`)
	if !ok {
		return nil, false
	}
	secret := temp.([]byte)
	dec, err := utils.Encrypt(data, secret)
	if err != nil {
		return nil, false
	}
	return dec, true
}

func Decrypt(data []byte, session *melody.Session) ([]byte, bool) {
	temp, ok := session.Get(`Secret`)
	if !ok {
		return nil, false
	}
	secret := temp.([]byte)
	dec, err := utils.Decrypt(data, secret)
	if err != nil {
		return nil, false
	}
	return dec, true
}

func GetRemoteAddr(ctx *gin.Context) string {
	if remote, ok := ctx.RemoteIP(); ok {
		if remote.IsLoopback() {
			forwarded := ctx.GetHeader(`X-Forwarded-For`)
			if len(forwarded) > 0 {
				return forwarded
			}
			realIP := ctx.GetHeader(`X-Real-IP`)
			if len(realIP) > 0 {
				return realIP
			}
		} else {
			if ip := remote.To4(); ip != nil {
				return ip.String()
			}
			if ip := remote.To16(); ip != nil {
				return ip.String()
			}
		}
	}

	remote := net.ParseIP(ctx.Request.RemoteAddr)
	if remote != nil {
		if remote.IsLoopback() {
			forwarded := ctx.GetHeader(`X-Forwarded-For`)
			if len(forwarded) > 0 {
				return forwarded
			}
			realIP := ctx.GetHeader(`X-Real-IP`)
			if len(realIP) > 0 {
				return realIP
			}
		} else {
			if ip := remote.To4(); ip != nil {
				return ip.String()
			}
			if ip := remote.To16(); ip != nil {
				return ip.String()
			}
		}
	}
	addr := ctx.Request.RemoteAddr
	if pos := strings.LastIndex(addr, `:`); pos > -1 {
		return strings.Trim(addr[:pos], `[]`)
	}
	return addr
}

func CheckClientReq(ctx *gin.Context) *melody.Session {
	secret, err := hex.DecodeString(ctx.GetHeader(`Secret`))
	if err != nil || len(secret) != 32 {
		return nil
	}
	var result *melody.Session = nil
	Melody.IterSessions(func(uuid string, s *melody.Session) bool {
		if val, ok := s.Get(`Secret`); ok {
			// Check if there's a connection matches this secret.
			if b, ok := val.([]byte); ok && bytes.Equal(b, secret) {
				result = s
				return false
			}
		}
		return true
	})
	return result
}

func CheckDevice(deviceID, connUUID string) (string, bool) {
	if len(connUUID) > 0 {
		if !Devices.Has(connUUID) {
			return connUUID, true
		}
	} else {
		tempConnUUID := ``
		Devices.IterCb(func(uuid string, v interface{}) bool {
			device := v.(*modules.Device)
			if device.ID == deviceID {
				tempConnUUID = uuid
				return false
			}
			return true
		})
		return tempConnUUID, len(tempConnUUID) > 0
	}
	return ``, false
}

func EncAES(data []byte, key []byte) ([]byte, error) {
	hash, _ := utils.GetMD5(data)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	stream := cipher.NewCTR(block, hash)
	encBuffer := make([]byte, len(data))
	stream.XORKeyStream(encBuffer, data)
	return append(hash, encBuffer...), nil
}

func DecAES(data []byte, key []byte) ([]byte, error) {
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

func RemoveBytesPrefix(data *[]byte, n int) *[]byte {
	sliceHeader := (*reflect.SliceHeader)(unsafe.Pointer(data))
	header := &reflect.SliceHeader{
		Data: sliceHeader.Data + uintptr(n),
		Len:  sliceHeader.Len - n,
		Cap:  sliceHeader.Cap - n,
	}
	return (*[]byte)(unsafe.Pointer(header))
}

func RemoveBytesSuffix(data *[]byte, n int) *[]byte {
	sliceHeader := (*reflect.SliceHeader)(unsafe.Pointer(data))
	header := &reflect.SliceHeader{
		Data: sliceHeader.Data,
		Len:  sliceHeader.Len - n,
		Cap:  sliceHeader.Cap - n,
	}
	return (*[]byte)(unsafe.Pointer(header))
}
