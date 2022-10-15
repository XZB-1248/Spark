package main

import (
	"Spark/modules"
	"Spark/server/auth"
	"Spark/server/common"
	"Spark/server/config"
	"Spark/server/handler"
	"Spark/server/handler/desktop"
	"Spark/server/handler/terminal"
	"Spark/server/handler/utility"
	"Spark/utils/cmap"
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"github.com/rakyll/statik/fs"
	"io"
	"net"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"

	_ "Spark/server/embed/web"
	"Spark/utils"
	"Spark/utils/melody"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kataras/golog"
)

var blocked = cmap.New()
var lastRequest = time.Now().Unix()

func main() {
	webFS, err := fs.NewWithNamespace(`web`)
	if err != nil {
		common.Fatal(nil, `LOAD_STATIC_RES`, `fail`, err.Error(), nil)
		return
	}
	gin.SetMode(gin.ReleaseMode)
	app := gin.New()
	app.Use(gin.Recovery())
	{
		handler.AuthHandler = checkAuth()
		handler.InitRouter(app.Group(`/api`))
		app.Any(`/ws`, wsHandshake)
		app.NoRoute(handler.AuthHandler, func(ctx *gin.Context) {
			if !serveGzip(ctx, webFS) && !checkCache(ctx, webFS) {
				http.FileServer(webFS).ServeHTTP(ctx.Writer, ctx.Request)
			}
		})
	}

	common.Melody.Config.MaxMessageSize = common.MaxMessageSize
	common.Melody.HandleConnect(wsOnConnect)
	common.Melody.HandleMessage(wsOnMessage)
	common.Melody.HandleMessageBinary(wsOnMessageBinary)
	common.Melody.HandleDisconnect(wsOnDisconnect)
	go wsHealthCheck(common.Melody)

	srv := &http.Server{
		Addr:    config.Config.Listen,
		Handler: app,
		ConnContext: func(ctx context.Context, c net.Conn) context.Context {
			ctx = context.WithValue(ctx, `Conn`, c)
			ctx = context.WithValue(ctx, `ClientIP`, common.GetAddrIP(c.RemoteAddr()))
			return ctx
		},
	}
	{
		go func() {
			err = srv.ListenAndServe()
		}()
		if err != nil {
			common.Fatal(nil, `SERVICE_INIT`, `fail`, err.Error(), nil)
		} else {
			common.Info(nil, `SERVICE_INIT`, ``, ``, map[string]any{
				`listen`: config.Config.Listen,
			})
		}
	}
	quit := make(chan os.Signal, 3)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	common.Warn(nil, `SERVICE_EXITING`, ``, ``, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		common.Warn(nil, `SERVICE_EXIT`, `error`, err.Error(), nil)
	}
	<-ctx.Done()
	common.Warn(nil, `SERVICE_EXIT`, `success`, ``, nil)
	common.CloseLog()
}

func wsHandshake(ctx *gin.Context) {
	if !ctx.IsWebsocket() {
		// When message is too large to transport via websocket,
		// client will try to send these data via http.
		const MaxBodySize = 2 << 18 // 524288 512KB
		if ctx.Request.ContentLength > MaxBodySize {
			ctx.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, modules.Packet{Code: 1})
			return
		}
		body, err := ctx.GetRawData()
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, modules.Packet{Code: 1})
			return
		}
		session := common.CheckClientReq(ctx)
		if session == nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, modules.Packet{Code: 1})
			return
		}
		wsOnMessageBinary(session, body)
		ctx.JSON(http.StatusOK, modules.Packet{Code: 0})
		return
	}

	clientUUID, _ := hex.DecodeString(ctx.GetHeader(`UUID`))
	clientKey, _ := hex.DecodeString(ctx.GetHeader(`Key`))
	if len(clientUUID) != 16 || len(clientKey) != 32 {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	decrypted, err := common.DecAES(clientKey, config.Config.SaltBytes)
	if err != nil || !bytes.Equal(decrypted, clientUUID) {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	secret := append(utils.GetUUID(), utils.GetUUID()...)
	err = common.Melody.HandleRequestWithKeys(ctx.Writer, ctx.Request, http.Header{
		`Secret`: []string{hex.EncodeToString(secret)},
	}, gin.H{
		`Secret`:   secret,
		`LastPack`: utils.Unix,
		`Address`:  common.GetRemoteAddr(ctx),
	})
	if err != nil {
		golog.Error(err)
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}
}

func wsOnConnect(session *melody.Session) {
	pingDevice(session)
}

func wsOnMessage(session *melody.Session, _ []byte) {
	session.Close()
}

func wsOnMessageBinary(session *melody.Session, data []byte) {
	var pack modules.Packet

	{
		dataLen := len(data)
		if dataLen >= 22 {
			if bytes.Equal(data[:5], []byte{34, 22, 19, 17, 20}) {
				event := hex.EncodeToString(data[6:22])
				copy(data[6:], data[22:])
				common.CallEvent(modules.Packet{
					Event: event,
					Data: gin.H{
						`data`: utils.GetSlicePrefix(&data, dataLen-16),
					},
				}, session)
				return
			}
		}
	}

	data, ok := common.Decrypt(data, session)
	if !(ok && utils.JSON.Unmarshal(data, &pack) == nil) {
		common.SendPack(modules.Packet{Code: -1}, session)
		session.CloseWithMsg(melody.FormatCloseMessage(1000, `invalid request`))
		return
	}
	if pack.Act == `report` || pack.Act == `setDevice` {
		session.Set(`LastPack`, utils.Unix)
		utility.OnDevicePack(data, session)
		return
	}
	if !common.Devices.Has(session.UUID) {
		session.CloseWithMsg(melody.FormatCloseMessage(1001, `invalid device id`))
		return
	}
	common.CallEvent(pack, session)
	session.Set(`LastPack`, utils.Unix)
}

func wsOnDisconnect(session *melody.Session) {
	if val, ok := common.Devices.Get(session.UUID); ok {
		deviceInfo := val.(*modules.Device)
		terminal.CloseSessionsByDevice(deviceInfo.ID)
		desktop.CloseSessionsByDevice(deviceInfo.ID)
		common.Info(nil, `CLIENT_OFFLINE`, ``, ``, map[string]any{
			`device`: map[string]any{
				`name`: deviceInfo.Hostname,
				`ip`:   deviceInfo.WAN,
			},
		})
	} else {
		common.Info(nil, `CLIENT_OFFLINE`, ``, ``, map[string]any{
			`device`: map[string]any{
				`ip`: common.GetAddrIP(session.GetWSConn().UnderlyingConn().RemoteAddr()),
			},
		})
	}
	common.Devices.Remove(session.UUID)
}

func wsHealthCheck(container *melody.Melody) {
	const MaxIdleSeconds = 150
	const MaxPingInterval = 60
	go func() {
		// Ping clients with a dynamic interval.
		// Interval will be greater than 3 seconds and less than MaxPingInterval.
		var tick int64 = 0
		var pingInterval int64 = 3
		for range time.NewTicker(3 * time.Second).C {
			tick += 3
			if tick >= utils.Unix-lastRequest {
				pingInterval = 3
			}
			if tick >= 3 && (tick >= pingInterval || tick >= MaxPingInterval) {
				pingInterval += 3
				if pingInterval > MaxPingInterval {
					pingInterval = MaxPingInterval
				}
				tick = 0
				container.IterSessions(func(uuid string, s *melody.Session) bool {
					go pingDevice(s)
					return true
				})
			}
		}
	}()
	for now := range time.NewTicker(60 * time.Second).C {
		timestamp := now.Unix()
		// Store sessions to be disconnected.
		queue := make([]*melody.Session, 0)
		container.IterSessions(func(uuid string, s *melody.Session) bool {
			val, ok := s.Get(`LastPack`)
			if !ok {
				queue = append(queue, s)
				return true
			}
			lastPack, ok := val.(int64)
			if !ok {
				queue = append(queue, s)
				return true
			}
			if timestamp-lastPack > MaxIdleSeconds {
				queue = append(queue, s)
			}
			return true
		})
		for i := 0; i < len(queue); i++ {
			queue[i].Close()
		}
	}
}

func pingDevice(s *melody.Session) {
	t := time.Now().UnixMilli()
	trigger := utils.GetStrUUID()
	common.SendPack(modules.Packet{Act: `ping`, Event: trigger}, s)
	common.AddEventOnce(func(packet modules.Packet, session *melody.Session) {
		val, ok := common.Devices.Get(s.UUID)
		if ok {
			deviceInfo := val.(*modules.Device)
			deviceInfo.Latency = uint(time.Now().UnixMilli()-t) / 2
		}
	}, s.UUID, trigger, 3*time.Second)
}

func checkAuth() gin.HandlerFunc {
	// Token as key and update timestamp as value.
	// Stores authenticated tokens.
	tokens := cmap.New()
	go func() {
		for now := range time.NewTicker(60 * time.Second).C {
			var queue []string
			tokens.IterCb(func(key string, v any) bool {
				if now.Unix()-v.(int64) > 1800 {
					queue = append(queue, key)
				}
				return true
			})
			tokens.Remove(queue...)
			queue = nil

			blocked.IterCb(func(addr string, v any) bool {
				if now.Unix() > v.(int64) {
					queue = append(queue, addr)
				}
				return true
			})
			blocked.Remove(queue...)
		}
	}()

	if config.Config.Auth == nil || len(config.Config.Auth) == 0 {
		return func(ctx *gin.Context) {
			lastRequest = utils.Unix
			ctx.Next()
		}
	}

	auth := auth.BasicAuth(config.Config.Auth, ``)
	return func(ctx *gin.Context) {
		now := utils.Unix
		passed := false

		if token, err := ctx.Cookie(`Authorization`); err == nil {
			if tokens.Has(token) {
				lastRequest = now
				tokens.Set(token, now)
				passed = true
				return
			}
		}

		if !passed {
			addr := common.GetRealIP(ctx)
			if expire, ok := blocked.Get(addr); ok {
				if now < expire.(int64) {
					ctx.AbortWithStatusJSON(http.StatusTooManyRequests, modules.Packet{Code: 1})
					return
				}
				blocked.Remove(addr)
			}

			auth(ctx)
			user := ctx.GetString(`user`)

			if ctx.IsAborted() {
				blocked.Set(addr, now+1)
				user = utils.If(len(user) == 0, `EMPTY`, user)
				common.Warn(ctx, `LOGIN_ATTEMPT`, `fail`, ``, map[string]any{
					`user`: user,
				})
				return
			}

			common.Warn(ctx, `LOGIN_ATTEMPT`, `success`, ``, map[string]any{
				`user`: user,
			})
			token := utils.GetStrUUID()
			tokens.Set(token, now)
			ctx.Header(`Set-Cookie`, fmt.Sprintf(`Authorization=%s; Path=/; HttpOnly`, token))
		}
		lastRequest = now
	}
}

func serveGzip(ctx *gin.Context, statikFS http.FileSystem) bool {
	headers := ctx.Request.Header
	filename := path.Clean(ctx.Request.RequestURI)
	if !strings.Contains(headers.Get(`Accept-Encoding`), `gzip`) {
		return false
	}
	if strings.Contains(headers.Get(`Connection`), `Upgrade`) {
		return false
	}
	if strings.Contains(headers.Get(`Accept`), `text/event-stream`) {
		return false
	}

	file, err := statikFS.Open(filename + `.gz`)
	if err != nil {
		return false
	}

	file.Seek(0, io.SeekStart)
	conn, ok := ctx.Request.Context().Value(`Conn`).(net.Conn)
	if !ok {
		file.Close()
		return false
	}

	etag := fmt.Sprintf(`"%x-%s"`, []byte(filename), config.COMMIT)
	if headers.Get(`If-None-Match`) == etag {
		ctx.Status(http.StatusNotModified)
		return true
	}
	ctx.Header(`Cache-Control`, `max-age=604800`)
	ctx.Header(`ETag`, etag)
	ctx.Header(`Expires`, utils.Now.Add(7*24*time.Hour).Format(`Mon, 02 Jan 2006 15:04:05 GMT`))

	ctx.Writer.Header().Del(`Content-Length`)
	ctx.Header(`Content-Encoding`, `gzip`)
	ctx.Header(`Vary`, `Accept-Encoding`)
	ctx.Status(http.StatusOK)

	for {
		eof := false
		buf := make([]byte, 2<<14)
		n, err := file.Read(buf)
		if n == 0 {
			break
		}
		if err != nil {
			eof = err == io.EOF
			if !eof {
				break
			}
		}
		conn.SetWriteDeadline(utils.Now.Add(10 * time.Second))
		_, err = ctx.Writer.Write(buf[:n])
		if eof || err != nil {
			break
		}
	}
	conn.SetWriteDeadline(time.Time{})
	file.Close()
	ctx.Done()
	return true
}

func checkCache(ctx *gin.Context, _ http.FileSystem) bool {
	filename := path.Clean(ctx.Request.RequestURI)

	etag := fmt.Sprintf(`"%x-%s"`, []byte(filename), config.COMMIT)
	if ctx.Request.Header.Get(`If-None-Match`) == etag {
		ctx.Status(http.StatusNotModified)
		return true
	}
	ctx.Header(`ETag`, etag)
	ctx.Header(`Cache-Control`, `max-age=604800`)
	ctx.Header(`Expires`, utils.Now.Add(7*24*time.Hour).Format(`Mon, 02 Jan 2006 15:04:05 GMT`))
	return false
}
