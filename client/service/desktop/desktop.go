package desktop

import (
	"Spark/client/common"
	"Spark/modules"
	"Spark/utils"
	"Spark/utils/cmap"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"github.com/kbinani/screenshot"
	"image"
	"image/jpeg"
	"reflect"
	"runtime"
	"sync"
	"time"
	"unsafe"
)

type session struct {
	lastPack int64
	rawEvent []byte
	event    string
	escape   bool
	channel  chan message
	lock     *sync.Mutex
}
type message struct {
	t     int
	info  string
	frame *[]*[]byte
}

// frame packet format:
// +---------+---------+----------+-------------+----------+---------+---------+---------+---------+-------+
// | magic   | op code | event id | body length | img type | x       | y       | width   | height  | image |
// +---------+---------+----------+-------------+----------+---------+---------+---------+---------+-------+
// | 5 bytes | 1 byte  | 16 bytes | 2 bytes     | 2 bytes  | 2 bytes | 2 bytes | 2 bytes | 2 bytes | -     |
// +---------+---------+----------+-------------+----------+---------+---------+---------+---------+-------+

// magic:
// []byte{34, 22, 19, 17, 20}

// op code:
// 00: first part of a frame, device -> browser
// 01: rest parts of a frame, device -> browser
// 02: set resolution of every frame, device -> browser
// 03: JSON string, server -> browser

// img type:
// 0: raw image
// 1: compressed image (jpeg)

const compress = 1
const fpsLimit = 24
const blockSize = 96
const frameBuffer = 3
const displayIndex = 0
const imageQuality = 70

var lock = &sync.Mutex{}
var working = false
var sessions = cmap.New()
var prevDesktop *image.RGBA
var displayBounds image.Rectangle
var errNoImage = errors.New(`DESKTOP.NO_IMAGE_YET`)

func init() {
	go healthCheck()
}

func worker() {
	runtime.LockOSThread()
	lock.Lock()
	if working {
		lock.Unlock()
		runtime.UnlockOSThread()
		return
	}
	working = true
	lock.Unlock()

	var (
		numErrors int
		screen    Screen
		img       *image.RGBA
		err       error
	)
	screen.Init(displayIndex, displayBounds)
	for working {
		if sessions.Count() == 0 {
			break
		}
		img, err = screen.Capture()
		if err != nil {
			if err == errNoImage {
				<-time.After(time.Second / fpsLimit)
				continue
			}
			numErrors++
			if numErrors > 10 {
				break
			}
		} else {
			numErrors = 0
			diff := imageCompare(img, prevDesktop, compress)
			if diff != nil && len(diff) > 0 {
				prevDesktop = img
				sendImageDiff(diff)
			}
			<-time.After(time.Second / fpsLimit)
		}
	}
	img = nil
	prevDesktop = nil
	if numErrors > 10 {
		quitAllDesktop(err.Error())
	}
	lock.Lock()
	working = false
	lock.Unlock()
	screen.Release()
	runtime.UnlockOSThread()
	go runtime.GC()
}

func sendImageDiff(diff []*[]byte) {
	sessions.IterCb(func(uuid string, t any) bool {
		desktop := t.(*session)
		desktop.lock.Lock()
		if !desktop.escape {
			if len(desktop.channel) >= frameBuffer {
				select {
				case <-desktop.channel:
				default:
				}
			}
			desktop.channel <- message{t: 0, frame: &diff}
		}
		desktop.lock.Unlock()
		return true
	})
}

func quitAllDesktop(info string) {
	keys := make([]string, 0)
	sessions.IterCb(func(uuid string, t any) bool {
		keys = append(keys, uuid)
		desktop := t.(*session)
		desktop.escape = true
		desktop.channel <- message{t: 1, info: info}
		return true
	})
	sessions.Clear()
	lock.Lock()
	working = false
	lock.Unlock()
}

func imageCompare(img, prev *image.RGBA, compress int) []*[]byte {
	result := make([]*[]byte, 0)
	if prev == nil {
		return splitFullImage(img, compress)
	}
	diff := getDiff(img, prev)
	if diff == nil {
		return result
	}
	for _, rect := range diff {
		block := getImageBlock(img, rect, compress)
		block = makeImageBlock(block, rect, compress)
		result = append(result, &block)
	}
	return result
}

func splitFullImage(img *image.RGBA, compress int) []*[]byte {
	if img == nil {
		return nil
	}
	result := make([]*[]byte, 0)
	rect := img.Rect
	imgWidth := rect.Dx()
	imgHeight := rect.Dy()
	for y := rect.Min.Y; y < rect.Max.Y; y += blockSize {
		height := utils.If(y+blockSize > imgHeight, imgHeight-y, blockSize)
		for x := rect.Min.X; x < rect.Max.X; x += blockSize {
			width := utils.If(x+blockSize > imgWidth, imgWidth-x, blockSize)
			blockRect := image.Rect(x, y, x+width, y+height)
			block := getImageBlock(img, blockRect, compress)
			block = makeImageBlock(block, blockRect, compress)
			result = append(result, &block)
		}
	}
	return result
}

func getImageBlock(img *image.RGBA, rect image.Rectangle, compress int) []byte {
	width := rect.Dx()
	height := rect.Dy()
	buf := make([]byte, width*height*4)
	bufPos := 0
	imgPos := img.PixOffset(rect.Min.X, rect.Min.Y)
	for y := 0; y < height; y++ {
		copy(buf[bufPos:bufPos+width*4], img.Pix[imgPos:imgPos+width*4])
		bufPos += width * 4
		imgPos += img.Stride
	}
	switch compress {
	case 0:
		return buf
	case 1:
		subImg := &image.RGBA{
			Pix:    buf,
			Stride: width * 4,
			Rect:   image.Rect(0, 0, width, height),
		}
		writer := &bytes.Buffer{}
		jpeg.Encode(writer, subImg, &jpeg.Options{Quality: imageQuality})
		return writer.Bytes()
	}
	return nil
}

func makeImageBlock(block []byte, rect image.Rectangle, compress int) []byte {
	buf := make([]byte, 12)
	binary.BigEndian.PutUint16(buf[0:2], uint16(len(block)+10))
	binary.BigEndian.PutUint16(buf[2:4], uint16(compress))
	binary.BigEndian.PutUint16(buf[4:6], uint16(rect.Min.X))
	binary.BigEndian.PutUint16(buf[6:8], uint16(rect.Min.Y))
	binary.BigEndian.PutUint16(buf[8:10], uint16(rect.Size().X))
	binary.BigEndian.PutUint16(buf[10:12], uint16(rect.Size().Y))
	buf = append(buf, block...)
	return buf
}

func getDiff(img, prev *image.RGBA) []image.Rectangle {
	imgWidth := img.Rect.Dx()
	imgHeight := img.Rect.Dy()
	result := make([]image.Rectangle, 0)
	for y := 0; y < imgHeight; y += blockSize * 2 {
		height := utils.If(y+blockSize > imgHeight, imgHeight-y, blockSize)
		for x := 0; x < imgWidth; x += blockSize {
			width := utils.If(x+blockSize > imgWidth, imgWidth-x, blockSize)
			rect := image.Rect(x, y, x+width, y+height)
			if isDiff(img, prev, rect) {
				result = append(result, rect)
			}
		}
	}
	for y := blockSize; y < imgHeight; y += blockSize * 2 {
		height := utils.If(y+blockSize > imgHeight, imgHeight-y, blockSize)
		for x := 0; x < imgWidth; x += blockSize {
			width := utils.If(x+blockSize > imgWidth, imgWidth-x, blockSize)
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
	imgWidth := img.Rect.Dx()
	rectWidth := rect.Dx()

	end := 0
	if rect.Max.Y == 0 {
		end = rect.Max.X * 4
	} else {
		end = (rect.Max.Y*imgWidth - imgWidth + rect.Max.X) * 4
	}
	if imgHeader.Len < end || prevHeader.Len < end {
		return true
	}
	for y := rect.Min.Y; y < rect.Max.Y; y += 2 {
		cursor := uintptr((y*imgWidth + rect.Min.X) * 4)
		for x := 0; x < rectWidth; x += 4 {
			if *(*uint64)(unsafe.Pointer(imgPtr + cursor)) != *(*uint64)(unsafe.Pointer(prevPtr + cursor)) {
				return true
			}
			cursor += 16
		}
	}
	return false
}

func InitDesktop(pack modules.Packet) error {
	var uuid string
	rawEvent, err := hex.DecodeString(pack.Event)
	if err != nil {
		return err
	}
	if val, ok := pack.GetData(`desktop`, reflect.String); !ok {
		return errors.New(`${i18n|COMMON.INVALID_PARAMETER}`)
	} else {
		uuid = val.(string)
	}
	desktop := &session{
		event:    pack.Event,
		rawEvent: rawEvent,
		lastPack: utils.Unix,
		escape:   false,
		channel:  make(chan message, 5),
		lock:     &sync.Mutex{},
	}
	{
		displayBounds = screenshot.GetDisplayBounds(displayIndex)
		if screenshot.NumActiveDisplays() == 0 {
			if displayBounds.Dx() == 0 || displayBounds.Dy() == 0 {
				close(desktop.channel)
				data, _ := utils.JSON.Marshal(modules.Packet{Act: `DESKTOP_QUIT`, Msg: `${i18n|DESKTOP.NO_DISPLAY_FOUND}`})
				data = utils.XOR(data, common.WSConn.GetSecret())
				common.WSConn.SendRawData(desktop.rawEvent, data, 20, 03)
				return errors.New(`${i18n|DESKTOP.NO_DISPLAY_FOUND}`)
			}
		}
		desktop.channel <- message{t: 2}
	}
	go handleDesktop(pack, uuid, desktop)
	if !working {
		sessions.Set(uuid, desktop)
		go worker()
	} else {
		img := splitFullImage(prevDesktop, compress)
		desktop.lock.Lock()
		desktop.channel <- message{t: 0, frame: &img}
		desktop.lock.Unlock()
		sessions.Set(uuid, desktop)
	}
	return nil
}

func PingDesktop(pack modules.Packet) {
	var uuid string
	var desktop *session
	if val, ok := pack.GetData(`desktop`, reflect.String); !ok {
		return
	} else {
		uuid = val.(string)
	}
	if val, ok := sessions.Get(uuid); ok {
		desktop = val.(*session)
		desktop.lastPack = utils.Unix
	}
}

func KillDesktop(pack modules.Packet) {
	var uuid string
	var desktop *session
	if val, ok := pack.GetData(`desktop`, reflect.String); !ok {
		return
	} else {
		uuid = val.(string)
	}
	if val, ok := sessions.Get(uuid); !ok {
		return
	} else {
		desktop = val.(*session)
	}
	sessions.Remove(uuid)
	data, _ := utils.JSON.Marshal(modules.Packet{Act: `DESKTOP_QUIT`, Msg: `${i18n|DESKTOP.SESSION_CLOSED}`})
	data = utils.XOR(data, common.WSConn.GetSecret())
	common.WSConn.SendRawData(desktop.rawEvent, data, 20, 03)
	desktop.lock.Lock()
	desktop.escape = true
	desktop.rawEvent = nil
	desktop.lock.Unlock()
}

func GetDesktop(pack modules.Packet) {
	var uuid string
	var desktop *session
	if val, ok := pack.GetData(`desktop`, reflect.String); !ok {
		return
	} else {
		uuid = val.(string)
	}
	if val, ok := sessions.Get(uuid); !ok {
		return
	} else {
		desktop = val.(*session)
	}
	if !desktop.escape {
		lock.Lock()
		img := splitFullImage(prevDesktop, compress)
		lock.Unlock()
		desktop.lock.Lock()
		desktop.channel <- message{t: 0, frame: &img}
		desktop.lock.Unlock()
	}
}

func handleDesktop(pack modules.Packet, uuid string, desktop *session) {
	for !desktop.escape {
		select {
		case msg, ok := <-desktop.channel:
			// send error info
			if msg.t == 1 || !ok {
				data, _ := utils.JSON.Marshal(modules.Packet{Act: `DESKTOP_QUIT`, Msg: msg.info})
				data = utils.XOR(data, common.WSConn.GetSecret())
				common.WSConn.SendRawData(desktop.rawEvent, data, 20, 03)
				desktop.escape = true
				sessions.Remove(uuid)
				break
			}
			// send image
			if msg.t == 0 {
				buf := append([]byte{34, 22, 19, 17, 20, 00}, desktop.rawEvent...)
				for _, slice := range *msg.frame {
					if len(buf)+len(*slice) >= common.MaxMessageSize {
						if common.WSConn.SendData(buf) != nil {
							break
						}
						buf = append([]byte{34, 22, 19, 17, 20, 01}, desktop.rawEvent...)
					}
					buf = append(buf, *slice...)
				}
				common.WSConn.SendData(buf)
				buf = nil
				continue
			}
			// set resolution
			if msg.t == 2 {
				buf := append([]byte{34, 22, 19, 17, 20, 02}, desktop.rawEvent...)
				data := make([]byte, 6)
				binary.BigEndian.PutUint16(data[:2], 4)
				binary.BigEndian.PutUint16(data[2:4], uint16(displayBounds.Dx()))
				binary.BigEndian.PutUint16(data[4:6], uint16(displayBounds.Dy()))
				buf = append(buf, data...)
				common.WSConn.SendData(buf)
				continue
			}
		case <-time.After(7 * time.Second):
			continue
		}
	}
}

func healthCheck() {
	const MaxInterval = 30
	for now := range time.NewTicker(30 * time.Second).C {
		timestamp := now.Unix()
		// stores sessions to be disconnected
		keys := make([]string, 0)
		sessions.IterCb(func(uuid string, t any) bool {
			desktop := t.(*session)
			if timestamp-desktop.lastPack > MaxInterval {
				keys = append(keys, uuid)
			}
			return true
		})
		sessions.Remove(keys...)
	}
}
