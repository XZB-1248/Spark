package main

import (
	"Spark/modules"
	"Spark/server/common"
	"Spark/server/config"
	"Spark/server/handler"
	"Spark/utils/cmap"
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"github.com/rakyll/statik/fs"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "Spark/server/embed/web"
	"Spark/utils"
	"Spark/utils/melody"
	"io/ioutil"
	"net/http"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/kataras/golog"
)

var lastRequest = time.Now().Unix()

func main() {
	golog.SetTimeFormat(`2006/01/02 15:04:05`)

	data, err := ioutil.ReadFile(`./Config.json`)
	if err != nil {
		golog.Fatal(`Failed to read config file: `, err)
		return
	}
	err = utils.JSON.Unmarshal(data, &config.Config)
	if err != nil {
		golog.Fatal(`Failed to parse config file: `, err)
		return
	}
	if len(config.Config.Salt) > 24 {
		golog.Fatal(`Length of Salt should be less than 24.`)
		return
	}
	config.Config.StdSalt = []byte(config.Config.Salt)
	config.Config.StdSalt = append(config.Config.StdSalt, bytes.Repeat([]byte{25}, 24)...)
	config.Config.StdSalt = config.Config.StdSalt[:24]

	webFS, err := fs.NewWithNamespace(`web`)
	if err != nil {
		golog.Fatal(`Failed to load static resources: `, err)
		return
	}
	if config.Config.Debug.Gin {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	app := gin.New()
	if config.Config.Debug.Pprof {
		pprof.Register(app)
	}
	{
		handler.AuthHandler = authCheck()
		handler.InitRouter(app.Group(`/api`))
		app.Any(`/ws`, wsHandshake)
		app.NoRoute(handler.AuthHandler, func(ctx *gin.Context) {
			http.FileServer(webFS).ServeHTTP(ctx.Writer, ctx.Request)
		})
	}

	common.Melody.Config.MaxMessageSize = 1024
	common.Melody.HandleConnect(wsOnConnect)
	common.Melody.HandleMessage(wsOnMessage)
	common.Melody.HandleMessageBinary(wsOnMessageBinary)
	common.Melody.HandleDisconnect(wsOnDisconnect)
	go wsHealthCheck(common.Melody)

	srv := &http.Server{Addr: config.Config.Listen, Handler: app}
	go func() {
		if err = srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			golog.Fatal(`Failed to bind address: `, err)
		}
	}()
	quit := make(chan os.Signal, 3)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	golog.Warn(`Server is shutting down ...`)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		golog.Fatal(`Server shutdown: `, err)
	}
	<-ctx.Done()
	golog.Info(`Server exited.`)
}

func wsHandshake(ctx *gin.Context) {
	if !ctx.IsWebsocket() {
		// When message is too large to transport via websocket,
		// client will try to send these data via http.
		const MaxBodySize = 2 << 18 //524288 512KB
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
	decrypted, err := common.DecAES(clientKey, config.Config.StdSalt)
	if err != nil || !bytes.Equal(decrypted, clientUUID) {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	secret := append(utils.GetUUID(), utils.GetUUID()...)
	err = common.Melody.HandleRequestWithKeys(ctx.Writer, ctx.Request, http.Header{
		`Secret`: []string{hex.EncodeToString(secret)},
	}, gin.H{
		`Secret`:   secret,
		`LastPack`: common.Unix,
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

func wsOnMessage(session *melody.Session, bytes []byte) {
	session.Close()
}

func wsOnMessageBinary(session *melody.Session, data []byte) {
	var pack modules.Packet
	data, ok := common.Decrypt(data, session)
	if !(ok && utils.JSON.Unmarshal(data, &pack) == nil) {
		common.SendPack(modules.Packet{Code: -1}, session)
		session.Close()
		return
	}
	if pack.Act == `report` || pack.Act == `setDevice` {
		session.Set(`LastPack`, common.Unix)
		handler.OnDevicePack(data, session)
		return
	}
	if !common.Devices.Has(session.UUID) {
		session.Close()
		return
	}
	common.CallEvent(pack, session)
	session.Set(`LastPack`, common.Unix)
}

func wsOnDisconnect(session *melody.Session) {
	if val, ok := common.Devices.Get(session.UUID); ok {
		deviceInfo := val.(*modules.Device)
		handler.CloseSessionsByDevice(deviceInfo.ID)
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
			if tick >= common.Unix-lastRequest {
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

func authCheck() gin.HandlerFunc {
	// Token as key and update timestamp as value.
	// Stores authenticated tokens.
	tokens := cmap.New()
	go func() {
		for now := range time.NewTicker(60 * time.Second).C {
			var queue []string
			tokens.IterCb(func(key string, v interface{}) bool {
				if now.Unix()-v.(int64) > 1800 {
					queue = append(queue, key)
				}
				return true
			})
			tokens.Remove(queue...)
		}
	}()

	auth := gin.BasicAuth(config.Config.Auth)
	return func(ctx *gin.Context) {
		now := common.Unix
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
			auth(ctx)
			if ctx.IsAborted() {
				return
			}
			token := utils.GetStrUUID()
			tokens.Set(token, now)
			ctx.Header(`Set-Cookie`, fmt.Sprintf(`Authorization=%s; Path=/; HttpOnly`, token))
		}
		lastRequest = now
	}
}
