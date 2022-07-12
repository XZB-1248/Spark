package desktop

import (
	"Spark/client/common"
	"Spark/modules"
	"Spark/utils/cmap"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"github.com/kbinani/screenshot"
	"image"
	"image/jpeg"
	"reflect"
	"sync"
	"time"
	"unsafe"
)

type session struct {
	lastPack int64
	binEvent []byte
	event    string
	escape   bool
	channel  chan message
	lock     *sync.Mutex
}
type message struct {
	t    int
	info string
	data *[][]byte
}

// +---------+---------+----------+----------+------------+---------+---------+---------+---------+-------+
// | magic   | op code | event id | img type | img length | x       | y       | width   | height  | image |
// +---------+---------+----------+----------+------------+---------+---------+---------+---------+-------+
// | 6 bytes | 1 byte  | 16 bytes | 2 bytes  | 2 bytes    | 2 bytes | 2 bytes | 2 bytes | 2 bytes | -     |
// +---------+---------+----------+----------+------------+---------+---------+---------+---------+-------+

// []byte{00, 22, 34, 19, 20}, magic bytes.

// Op code:
// 00: first part of a frame.
// 01: rest parts of a frame.
// 02: set resolution of every frame.
// 03: JSON string format. (Only for server).

// img type: 0: raw image, 1: compressed image (jpeg).

const compress = true
const blockSize = 64
const imgQuality = 80

var lock = &sync.Mutex{}
var working = false
var sessions = cmap.New()
var prevDesktop *image.RGBA

func init() {
	go healthCheck()
}

func worker() {
	lock.Lock()
	if working {
		lock.Unlock()
		return
	}
	working = true
	lock.Unlock()
	var (
		img    *image.RGBA
		err    error
		errors int
	)
	for working {
		if sessions.Count() == 0 {
			lock.Lock()
			working = false
			lock.Unlock()
			break
		}
		time.Sleep(30 * time.Millisecond)
		img, err = screenshot.CaptureDisplay(0)
		if err != nil {
			errors++
			if errors > 10 {
				break
			}
		} else {
			errors = 0
			diff := imageCompare(img, prevDesktop, compress)
			if diff != nil && len(diff) > 0 {
				prevDesktop = img
				sessions.IterCb(func(uuid string, t interface{}) bool {
					desktopSession := t.(*session)
					desktopSession.lock.Lock()
					if !desktopSession.escape {
						desktopSession.channel <- message{t: 0, data: &diff}
					}
					desktopSession.lock.Unlock()
					return true
				})
			}
		}
	}
	prevDesktop = nil
	if errors > 10 {
		quitAll(err.Error())
	}
	lock.Lock()
	working = false
	lock.Unlock()
}

func quitAll(info string) {
	keys := make([]string, 0)
	sessions.IterCb(func(uuid string, t interface{}) bool {
		keys = append(keys, uuid)
		desktopSession := t.(*session)
		desktopSession.escape = true
		desktopSession.channel <- message{t: 1, info: info}
		return true
	})
	sessions.Clear()
	lock.Lock()
	working = false
	lock.Unlock()
}

func imageCompare(img, prev *image.RGBA, compress bool) [][]byte {
	result := make([][]byte, 0)
	if prev == nil {
		return splitFullImage(img, compress)
	}
	diff := getDiff(img, prev)
	if diff == nil {
		return result
	}
	for _, rect := range diff {
		block := getImageBlock(img, rect, compress)
		buf := make([]byte, 12)
		if compress {
			binary.BigEndian.PutUint16(buf[0:2], uint16(1))
		} else {
			binary.BigEndian.PutUint16(buf[0:2], uint16(0))
		}
		binary.BigEndian.PutUint16(buf[2:4], uint16(len(block)))
		binary.BigEndian.PutUint16(buf[4:6], uint16(rect.Min.X))
		binary.BigEndian.PutUint16(buf[6:8], uint16(rect.Min.Y))
		binary.BigEndian.PutUint16(buf[8:10], uint16(rect.Size().X))
		binary.BigEndian.PutUint16(buf[10:12], uint16(rect.Size().Y))
		result = append(result, append(buf, block...))
	}
	return result
}

func splitFullImage(img *image.RGBA, compress bool) [][]byte {
	if img == nil {
		return nil
	}
	result := make([][]byte, 0)
	rect := img.Bounds()
	imgWidth := rect.Dx()
	imgHeight := rect.Dy()
	for y := rect.Min.Y; y < rect.Max.Y; y += blockSize {
		for x := rect.Min.X; x < rect.Max.X; x += blockSize {
			width := blockSize
			height := blockSize
			if x+width > imgWidth {
				width = imgWidth - x
			}
			if y+height > imgHeight {
				height = imgHeight - y
			}
			block := getImageBlock(img, image.Rect(x, y, x+width, y+height), compress)
			buf := make([]byte, 12)
			if compress {
				binary.BigEndian.PutUint16(buf[0:2], uint16(1))
			} else {
				binary.BigEndian.PutUint16(buf[0:2], uint16(0))
			}
			binary.BigEndian.PutUint16(buf[2:4], uint16(len(block)))
			binary.BigEndian.PutUint16(buf[4:6], uint16(x))
			binary.BigEndian.PutUint16(buf[6:8], uint16(y))
			binary.BigEndian.PutUint16(buf[8:10], uint16(width))
			binary.BigEndian.PutUint16(buf[10:12], uint16(height))
			result = append(result, append(buf, block...))
		}
	}
	return result
}

func getImageBlock(img *image.RGBA, rect image.Rectangle, compress bool) []byte {
	width := rect.Dx()
	height := rect.Dy()
	if rect.Min.X+width > img.Rect.Max.X {
		width = img.Rect.Max.X - rect.Min.X
	}
	if rect.Min.Y+height > img.Rect.Max.Y {
		height = img.Rect.Max.Y - rect.Min.Y
	}
	buf := make([]byte, 0)
	for y := 0; y < rect.Dy(); y++ {
		pos := (rect.Min.Y+y)*img.Rect.Size().X + rect.Min.X
		end := pos + width
		buf = append(buf, img.Pix[pos*4:end*4]...)

	}
	if !compress {
		return buf
	}
	newRect := image.Rect(0, 0, width, height)
	newImg := image.NewRGBA(newRect)
	copy(newImg.Pix[:len(buf)], buf[:])
	writer := new(bytes.Buffer)
	jpeg.Encode(writer, newImg, &jpeg.Options{Quality: imgQuality})
	return writer.Bytes()
}

func getDiff(img, prev *image.RGBA) []image.Rectangle {
	result := make([]image.Rectangle, 0)
	for y := 0; y < img.Rect.Size().Y; y += blockSize {
		for x := 0; x < img.Rect.Size().X; x += blockSize {
			width := blockSize
			height := blockSize
			if x+width > img.Rect.Size().X {
				width = img.Rect.Size().X - x
			}
			if y+height > img.Rect.Size().Y {
				height = img.Rect.Size().Y - y
			}
			rect := image.Rect(x, y, x+width, y+height)
			if isDiff(img, prev, rect) {
				result = append(result, rect)
			}
		}
	}
	return result
}

func isDiff(img, prev *image.RGBA, rect image.Rectangle) bool {
	imgHeader := (*reflect.SliceHeader)(unsafe.Pointer(&img.Pix))
	prevHeader := (*reflect.SliceHeader)(unsafe.Pointer(&prev.Pix))
	imgPtr := imgHeader.Data
	prevPtr := prevHeader.Data
	imgWidth := img.Rect.Size().X
	rectWidth := rect.Size().X

	end := 0
	if rect.Max.Y == 0 {
		end = rect.Max.X * 4
	} else {
		end = (rect.Max.Y*imgWidth - imgWidth + rect.Max.X) * 4
	}
	if imgHeader.Len < end || prevHeader.Len < end {
		return true
	}
	if rectWidth%2 == 0 {
		for y := rect.Min.Y; y < rect.Max.Y; y++ {
			cursor := uintptr((y*imgWidth + rect.Min.X) * 4)
			for x := 0; x < rectWidth; x += 2 {
				if *(*uint64)(unsafe.Pointer(imgPtr + cursor)) != *(*uint64)(unsafe.Pointer(prevPtr + cursor)) {
					return true
				}
				cursor += 8
			}
		}
	} else {
		for y := rect.Min.Y; y < rect.Max.Y; y++ {
			cursor := uintptr((y*imgWidth + rect.Min.X) * 4)
			for x := 0; x < rectWidth; x++ {
				if *(*uint32)(unsafe.Pointer(imgPtr + cursor)) != *(*uint32)(unsafe.Pointer(prevPtr + cursor)) {
					return true
				}
				cursor += 4
			}
		}
	}
	return false
}

func InitDesktop(pack modules.Packet) error {
	var desktop string
	binEvent, err := hex.DecodeString(pack.Event)
	if err != nil {
		return err
	}
	if val, ok := pack.GetData(`desktop`, reflect.String); !ok {
		return errors.New(`${i18n|invalidParameter}`)
	} else {
		desktop = val.(string)
	}
	desktopSession := &session{
		event:    pack.Event,
		binEvent: binEvent,
		lastPack: time.Now().Unix(),
		escape:   false,
		channel:  make(chan message, 4),
		lock:     &sync.Mutex{},
	}
	{
		// set resolution of desktop.
		if screenshot.NumActiveDisplays() == 0 {
			common.SendCb(modules.Packet{Act: `quitDesktop`, Msg: `${i18n|noDisplayFound}`}, pack, common.WSConn)
			return errors.New(`${i18n|noDisplayFound}`)
		}
		buf := append([]byte{00, 22, 34, 19, 20, 02}, binEvent...)
		data := make([]byte, 4)
		rect := screenshot.GetDisplayBounds(0)
		binary.BigEndian.PutUint16(data[:2], uint16(rect.Dx()))
		binary.BigEndian.PutUint16(data[2:], uint16(rect.Dy()))
		buf = append(buf, data...)
		common.SendData(buf, common.WSConn)
	}
	go func() {
		for !desktopSession.escape {
			select {
			case msg, ok := <-desktopSession.channel:
				// send error info
				if msg.t == 1 || !ok {
					common.SendCb(modules.Packet{Act: `quitDesktop`, Msg: msg.info}, pack, common.WSConn)
					desktopSession.escape = true
					sessions.Remove(desktop)
					break
				}
				// send image
				if msg.t == 0 {
					buf := append([]byte{00, 22, 34, 19, 20, 00}, binEvent...)
					for _, slice := range *msg.data {
						if len(buf)+len(slice) >= common.MaxMessageSize {
							if common.SendData(buf, common.WSConn) != nil {
								break
							}
							buf = append([]byte{00, 22, 34, 19, 20, 01}, binEvent...)
						}
						buf = append(buf, slice...)
					}
					common.SendData(buf, common.WSConn)
					buf = nil
					continue
				}
			case <-time.After(time.Second * 5):
			default:
				time.Sleep(50 * time.Millisecond)
			}
		}
	}()
	if !working {
		sessions.Set(desktop, desktopSession)
		go worker()
	} else {
		img := splitFullImage(prevDesktop, compress)
		desktopSession.lock.Lock()
		desktopSession.channel <- message{t: 0, data: &img}
		desktopSession.lock.Unlock()
		sessions.Set(desktop, desktopSession)
	}
	return nil
}

func PingDesktop(pack modules.Packet) {
	var desktop string
	var desktopSession *session
	if val, ok := pack.GetData(`desktop`, reflect.String); !ok {
		return
	} else {
		desktop = val.(string)
	}
	if val, ok := sessions.Get(desktop); !ok {
		return
	} else {
		desktopSession = val.(*session)
		desktopSession.lastPack = time.Now().Unix()
	}
}

func KillDesktop(pack modules.Packet) {
	var desktop string
	var desktopSession *session
	if val, ok := pack.GetData(`desktop`, reflect.String); !ok {
		return
	} else {
		desktop = val.(string)
	}
	if val, ok := sessions.Get(desktop); !ok {
		return
	} else {
		desktopSession = val.(*session)
	}
	sessions.Remove(desktop)
	desktopSession.lock.Lock()
	desktopSession.escape = true
	desktopSession.binEvent = nil
	desktopSession.lock.Unlock()
	common.SendCb(modules.Packet{Act: `quitDesktop`, Msg: `${i18n|desktopSessionClosed}`}, pack, common.WSConn)
}

func GetDesktop(pack modules.Packet) {
	var desktop string
	var desktopSession *session
	if val, ok := pack.GetData(`desktop`, reflect.String); !ok {
		return
	} else {
		desktop = val.(string)
	}
	if val, ok := sessions.Get(desktop); !ok {
		return
	} else {
		desktopSession = val.(*session)
	}
	if !desktopSession.escape {
		img := splitFullImage(prevDesktop, compress)
		desktopSession.lock.Lock()
		desktopSession.channel <- message{t: 0, data: &img}
		desktopSession.lock.Unlock()
	}
}

func healthCheck() {
	const MaxInterval = 30
	for now := range time.NewTicker(30 * time.Second).C {
		timestamp := now.Unix()
		// stores sessions to be disconnected
		keys := make([]string, 0)
		sessions.IterCb(func(uuid string, t interface{}) bool {
			desktopSession := t.(*session)
			if timestamp-desktopSession.lastPack > MaxInterval {
				keys = append(keys, uuid)
			}
			return true
		})
		sessions.Remove(keys...)
	}
}
