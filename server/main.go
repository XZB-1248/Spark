package main

import (
	"Spark/modules"
	"Spark/server/common"
	"Spark/server/config"
	"Spark/server/handler"
	"bytes"
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rakyll/statik/fs"

	_ "Spark/server/embed/built"
	_ "Spark/server/embed/web"
	"Spark/utils"
	"Spark/utils/melody"
	"encoding/hex"
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kataras/golog"
)

func main() {
	golog.SetTimeFormat(`2006/01/02 15:04:05`)
	gin.SetMode(`release`)

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
	common.BuiltFS, err = fs.NewWithNamespace(`built`)
	if err != nil {
		golog.Fatal(`Failed to load prebuilt clients: `, err)
		return
	}
	app := gin.New()
	{
		auth := gin.BasicAuth(config.Config.Auth)
		handler.InitRouter(app.Group(`/api`), auth)
		app.Any(`/ws`, wsHandshake)
		app.NoRoute(auth, func(ctx *gin.Context) {
			http.FileServer(webFS).ServeHTTP(ctx.Writer, ctx.Request)
		})
	}

	common.Melody.Config.MaxMessageSize = 1024
	common.Melody.HandleConnect(wsOnConnect)
	common.Melody.HandleMessage(wsOnMessage)
	common.Melody.HandleMessageBinary(wsOnMessageBinary)
	common.Melody.HandleDisconnect(wsOnDisconnect)
	go common.HealthCheckWS(90, common.Melody)

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
	if ctx.IsWebsocket() {
		clientUUID, _ := hex.DecodeString(ctx.GetHeader(`UUID`))
		clientKey, _ := hex.DecodeString(ctx.GetHeader(`Key`))
		if len(clientUUID) != 16 || len(clientKey) != 32 {
			ctx.Status(http.StatusUnauthorized)
			return
		}
		decrypted, err := common.DecAES(clientKey, config.Config.StdSalt)
		if err != nil || !bytes.Equal(decrypted, clientUUID) {
			ctx.Status(http.StatusUnauthorized)
			return
		}
		secret := append(utils.GetUUID(), utils.GetUUID()...)
		err = common.Melody.HandleRequestWithKeys(ctx.Writer, ctx.Request, http.Header{
			`Secret`: []string{hex.EncodeToString(secret)},
		}, gin.H{
			`Secret`:   secret,
			`LastPack`: time.Now().Unix(),
			`Address`:  common.GetRemoteAddr(ctx),
		})
		if err != nil {
			golog.Error(err)
			ctx.Status(http.StatusUpgradeRequired)
			return
		}
	} else {
		// When message is too large to transport via websocket,
		// client will try to send these data via http.
		const MaxBodySize = 2 << 18 //524288 512KB
		if ctx.Request.ContentLength > MaxBodySize {
			ctx.JSON(http.StatusRequestEntityTooLarge, modules.Packet{Code: 1})
			return
		}
		body, err := ctx.GetRawData()
		if err != nil {
			ctx.JSON(http.StatusBadRequest, modules.Packet{Code: 1})
			return
		}
		session := common.CheckClientReq(ctx)
		if session == nil {
			ctx.JSON(http.StatusUnauthorized, modules.Packet{Code: 1})
			return
		}
		wsOnMessageBinary(session, body)
		ctx.JSON(http.StatusOK, modules.Packet{Code: 0})
	}
}

func wsOnConnect(session *melody.Session) {
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
		session.Set(`LastPack`, time.Now().Unix())
		handler.OnDevicePack(data, session)
		return
	}
	if !common.Devices.Has(session.UUID) {
		session.Close()
		return
	}
	common.CallEvent(pack, session)
	session.Set(`LastPack`, time.Now().Unix())
}

func wsOnDisconnect(session *melody.Session) {
	if val, ok := common.Devices.Get(session.UUID); ok {
		deviceInfo := val.(*modules.Device)
		handler.CloseSessionsByDevice(deviceInfo.ID)
	}
	common.Devices.Remove(session.UUID)

}
