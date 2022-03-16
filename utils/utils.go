package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"errors"
	jsoniter "github.com/json-iterator/go"
)

var (
	ErrUnsupported        = errors.New(`unsupported operation`)
	ErrEntityInvalid      = errors.New(`entity is not valid`)
	ErrFailedVerification = errors.New(`failed to verify entity`)
	JSON                  = jsoniter.ConfigCompatibleWithStandardLibrary
)
func GenRandByte(n int) []byte {
	secBuffer := make([]byte, n)
	rand.Reader.Read(secBuffer)
	return secBuffer
}

func GetStrUUID() string {
	return hex.EncodeToString(GenRandByte(16))
}

func GetUUID() []byte {
	return GenRandByte(16)
}

func GetMD5(data []byte) ([]byte, string) {
	hash := md5.New()
	hash.Write(data)
	result := hash.Sum(nil)
	return result, hex.EncodeToString(result)
}

func Encrypt(data []byte, key []byte) ([]byte, error) {
	//fmt.Println(`Send: `, string(data))

	nonce := make([]byte, 64)
	rand.Reader.Read(nonce)
	data = append(data, nonce...)

	hash, _ := GetMD5(data)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	stream := cipher.NewCTR(block, hash)
	encBuffer := make([]byte, len(data))
	stream.XORKeyStream(encBuffer, data)
	return append(hash, encBuffer...), nil
}

func Decrypt(data []byte, key []byte) ([]byte, error) {
	// MD5[16 bytes] + Data[n bytes] + Nonce[64 bytes]
	dataLen := len(data)
	if dataLen <= 16+64 {
		return nil, ErrEntityInvalid
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	stream := cipher.NewCTR(block, data[:16])
	decBuffer := make([]byte, dataLen-16)
	stream.XORKeyStream(decBuffer, data[16:])

	hash, _ := GetMD5(decBuffer)
	if !bytes.Equal(hash, data[:16]) {
		return nil, ErrFailedVerification
	}

	//fmt.Println(`Recv: `, string(decBuffer[:dataLen-16-64]))
	return decBuffer[:dataLen-16-64], nil
}
