package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"reflect"
	"unsafe"
)

var (
	ErrEntityInvalid      = errors.New(`common.ENTITY_INVALID`)
	ErrFailedVerification = errors.New(`common.ENTITY_CHECK_FAILED`)
	JSON                  = jsoniter.ConfigCompatibleWithStandardLibrary
)

func If[T any](b bool, t, f T) T {
	if b {
		return t
	}
	return f
}

func Min[T int | int32 | int64 | uint | uint32 | uint64 | float32 | float64](a, b T) T {
	if a < b {
		return a
	}
	return b
}

func Max[T int | int32 | int64 | uint | uint32 | uint64 | float32 | float64](a, b T) T {
	if a > b {
		return a
	}
	return b
}

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
	hash.Reset()
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
		data = nil
		decBuffer = nil
		return nil, ErrFailedVerification
	}
	data = nil
	decBuffer = decBuffer[:dataLen-16-64]

	//fmt.Println(`Recv: `, string(decBuffer[:dataLen-16-64]))
	return decBuffer, nil
}

func FormatSize(size int64) string {
	sizes := []string{`B`, `KB`, `MB`, `GB`, `TB`, `PB`, `EB`, `ZB`, `YB`}
	i := 0
	for size >= 1024 && i < len(sizes)-1 {
		size /= 1024
		i++
	}
	return fmt.Sprintf(`%d%s`, size, sizes[i])
}

func BytesToString(b []byte, r ...int) string {
	sh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	bytesPtr := sh.Data
	bytesLen := sh.Len
	switch len(r) {
	case 1:
		r[0] = If(r[0] > bytesLen, bytesLen, r[0])
		bytesLen -= r[0]
		bytesPtr += uintptr(r[0])
	case 2:
		r[0] = If(r[0] > bytesLen, bytesLen, r[0])
		bytesLen = If(r[1] > bytesLen, bytesLen, r[1]) - r[0]
		bytesPtr += uintptr(r[0])
	}
	return *(*string)(unsafe.Pointer(&reflect.StringHeader{
		Data: bytesPtr,
		Len:  bytesLen,
	}))
}

func StringToBytes(s string, r ...int) []byte {
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	strPtr := sh.Data
	strLen := sh.Len
	switch len(r) {
	case 1:
		r[0] = If(r[0] > strLen, strLen, r[0])
		strLen -= r[0]
		strPtr += uintptr(r[0])
	case 2:
		r[0] = If(r[0] > strLen, strLen, r[0])
		strLen = If(r[1] > strLen, strLen, r[1]) - r[0]
		strPtr += uintptr(r[0])
	}
	return *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
		Data: strPtr,
		Len:  strLen,
		Cap:  strLen,
	}))
}

func GetSlicePrefix[T any](data *[]T, n int) *[]T {
	sliceHeader := (*reflect.SliceHeader)(unsafe.Pointer(data))
	return (*[]T)(unsafe.Pointer(&reflect.SliceHeader{
		Data: sliceHeader.Data,
		Len:  n,
		Cap:  n,
	}))
}

func GetSliceSuffix[T any](data *[]T, n int) *[]T {
	sliceHeader := (*reflect.SliceHeader)(unsafe.Pointer(data))
	return (*[]T)(unsafe.Pointer(&reflect.SliceHeader{
		Data: sliceHeader.Data + uintptr(sliceHeader.Len-n),
		Len:  n,
		Cap:  n,
	}))
}

func GetSliceChunk[T any](data *[]T, start, end int) *[]T {
	sliceHeader := (*reflect.SliceHeader)(unsafe.Pointer(data))
	return (*[]T)(unsafe.Pointer(&reflect.SliceHeader{
		Data: sliceHeader.Data + uintptr(start),
		Len:  end - start,
		Cap:  end - start,
	}))
}
