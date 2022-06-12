package handler

import (
	"Spark/modules"
	"Spark/server/common"
	"Spark/utils/cmap"
	"github.com/gin-gonic/gin"
	"github.com/kataras/golog"
	"io"
	"net"
	"net/http"
	"sync"
	"time"
)

// Bridge is a utility that handles the binary flow from the client
// to the browser or flow from the browser to the client.

type bridge struct {
	creation int64
	using    bool
	uuid     string
	lock     *sync.Mutex
	dst      *gin.Context
	src      *gin.Context
	ext      interface{}
	OnPull   func(bridge *bridge)
	OnPush   func(bridge *bridge)
	OnFinish func(bridge *bridge)
}

var bridges = cmap.New()

func init() {
	go func() {
		for now := range time.NewTicker(15 * time.Second).C {
			var queue []string
			timestamp := now.Unix()
			bridges.IterCb(func(k string, v interface{}) bool {
				b := v.(*bridge)
				if timestamp-b.creation > 60 && !b.using {
					b.lock.Lock()
					if b.src != nil && b.src.Request.Body != nil {
						b.src.Request.Body.Close()
					}
					b.src = nil
					b.dst = nil
					b.lock.Unlock()
					b = nil
					queue = append(queue, b.uuid)
				}
				return true
			})
			bridges.Remove(queue...)
		}
	}()
}

func checkBridge(ctx *gin.Context) *bridge {
	var form struct {
		Bridge string `json:"bridge" yaml:"bridge" form:"bridge" binding:"required"`
	}
	if err := ctx.ShouldBind(&form); err != nil {
		golog.Error(err)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, modules.Packet{Code: -1, Msg: `${i18n|invalidParameter}`})
		return nil
	}
	val, ok := bridges.Get(form.Bridge)
	if !ok {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, modules.Packet{Code: -1, Msg: `${i18n|invalidBridgeID}`})
		return nil
	}
	return val.(*bridge)
}

func bridgePush(ctx *gin.Context) {
	bridge := checkBridge(ctx)
	if bridge == nil {
		return
	}
	bridge.lock.Lock()
	if bridge.using || (bridge.src != nil && bridge.dst != nil) {
		bridge.lock.Unlock()
		ctx.AbortWithStatusJSON(http.StatusBadRequest, modules.Packet{Code: 1, Msg: `${i18n|bridgeAlreadyInUse}`})
		return
	}
	bridge.src = ctx
	bridge.using = true
	bridge.lock.Unlock()
	if bridge.OnPush != nil {
		bridge.OnPush(bridge)
	}
	if bridge.dst != nil && bridge.dst.Writer != nil {
		// Get net.Conn to set deadline manually.
		srcConn, srcOK := bridge.src.Request.Context().Value(`Conn`).(net.Conn)
		dstConn, dstOK := bridge.dst.Request.Context().Value(`Conn`).(net.Conn)
		if srcOK && dstOK {
			for {
				eof := false
				buf := make([]byte, 2<<14)
				srcConn.SetReadDeadline(common.Now.Add(5 * time.Second))
				n, err := bridge.src.Request.Body.Read(buf)
				if err != nil {
					eof = err == io.EOF
					if !eof {
						break
					}
				}
				if n == 0 {
					break
				}
				dstConn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				_, err = bridge.dst.Writer.Write(buf[:n])
				if eof || err != nil {
					break
				}
			}
		}
		srcConn.SetReadDeadline(time.Time{})
		dstConn.SetWriteDeadline(time.Time{})
		bridge.src.Status(http.StatusOK)
		if bridge.OnFinish != nil {
			bridge.OnFinish(bridge)
		}
		removeBridge(bridge.uuid)
		bridge = nil
	}
}

func bridgePull(ctx *gin.Context) {
	bridge := checkBridge(ctx)
	if bridge == nil {
		return
	}
	bridge.lock.Lock()
	if bridge.using || (bridge.src != nil && bridge.dst != nil) {
		bridge.lock.Unlock()
		ctx.AbortWithStatusJSON(http.StatusBadRequest, modules.Packet{Code: 1, Msg: `${i18n|bridgeAlreadyInUse}`})
		return
	}
	bridge.dst = ctx
	bridge.using = true
	bridge.lock.Unlock()
	if bridge.OnPull != nil {
		bridge.OnPull(bridge)
	}
	if bridge.src != nil && bridge.src.Request.Body != nil {
		// Get net.Conn to set deadline manually.
		srcConn, srcOK := bridge.src.Request.Context().Value(`Conn`).(net.Conn)
		dstConn, dstOK := bridge.dst.Request.Context().Value(`Conn`).(net.Conn)
		if srcOK && dstOK {
			for {
				eof := false
				buf := make([]byte, 2<<14)
				srcConn.SetReadDeadline(common.Now.Add(5 * time.Second))
				n, err := bridge.src.Request.Body.Read(buf)
				if err != nil {
					eof = err == io.EOF
					if !eof {
						break
					}
				}
				if n == 0 {
					break
				}
				dstConn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				_, err = bridge.dst.Writer.Write(buf[:n])
				if eof || err != nil {
					break
				}
			}
		}
		srcConn.SetReadDeadline(time.Time{})
		dstConn.SetWriteDeadline(time.Time{})
		bridge.src.Status(http.StatusOK)
		if bridge.OnFinish != nil {
			bridge.OnFinish(bridge)
		}
		removeBridge(bridge.uuid)
		bridge = nil
	}
}

func addBridge(ext interface{}, uuid string) *bridge {
	bridge := &bridge{
		creation: common.Unix,
		uuid:     uuid,
		using:    false,
		lock:     &sync.Mutex{},
		ext:      ext,
	}
	bridges.Set(uuid, bridge)
	return bridge
}

func addBridgeWithSrc(ext interface{}, uuid string, src *gin.Context) *bridge {
	bridge := &bridge{
		creation: common.Unix,
		uuid:     uuid,
		using:    false,
		lock:     &sync.Mutex{},
		ext:      ext,
		src:      src,
	}
	bridges.Set(uuid, bridge)
	return bridge
}

func addBridgeWithDst(ext interface{}, uuid string, dst *gin.Context) *bridge {
	bridge := &bridge{
		creation: common.Unix,
		uuid:     uuid,
		using:    false,
		lock:     &sync.Mutex{},
		ext:      ext,
		dst:      dst,
	}
	bridges.Set(uuid, bridge)
	return bridge
}

func removeBridge(uuid string) {
	val, ok := bridges.Get(uuid)
	if !ok {
		return
	}
	bridges.Remove(uuid)
	b := val.(*bridge)
	if b.src != nil && b.src.Request.Body != nil {
		b.src.Request.Body.Close()
	}
	b.src = nil
	b.dst = nil
	b = nil
}
