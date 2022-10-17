package common

import (
	"Spark/modules"
	"Spark/server/config"
	"Spark/utils"
	"Spark/utils/melody"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/kataras/golog"
	"io"
	"os"
	"time"
)

var logWriter *os.File
var disposed bool

func init() {
	setLogDst := func() {
		var err error
		if logWriter != nil {
			logWriter.Close()
		}
		if config.Config.Log.Level == `disable` || disposed {
			golog.SetOutput(os.Stdout)
			return
		}
		os.Mkdir(config.Config.Log.Path, 0666)
		now := utils.Now.Add(time.Second)
		logFile := fmt.Sprintf(`%s/%s.log`, config.Config.Log.Path, now.Format(`2006-01-02`))
		logWriter, err = os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
		if err != nil {
			golog.Warn(getLog(nil, `LOG_INIT`, `fail`, err.Error(), nil))
		}
		golog.SetOutput(io.MultiWriter(os.Stdout, logWriter))

		staleDate := time.Unix(now.Unix()-int64(config.Config.Log.Days*86400)-86400, 0)
		staleLog := fmt.Sprintf(`%s/%s.log`, config.Config.Log.Path, staleDate.Format(`2006-01-02`))
		os.Remove(staleLog)
	}
	setLogDst()
	go func() {
		waitSecs := 86400 - (utils.Now.Hour()*3600 + utils.Now.Minute()*60 + utils.Now.Second())
		<-time.After(time.Duration(waitSecs) * time.Second)
		setLogDst()
		for range time.NewTicker(time.Second * 86400).C {
			setLogDst()
		}
	}()
}

func getLog(ctx any, event, status, msg string, args map[string]any) string {
	if args == nil {
		args = map[string]any{}
	}
	args[`event`] = event
	if len(msg) > 0 {
		args[`msg`] = msg
	}
	if len(status) > 0 {
		args[`status`] = status
	}
	if ctx != nil {
		var connUUID string
		var targetInfo bool
		switch ctx.(type) {
		case *gin.Context:
			c := ctx.(*gin.Context)
			args[`from`] = GetRealIP(c)
			connUUID, targetInfo = c.Request.Context().Value(`ConnUUID`).(string)
		case *melody.Session:
			s := ctx.(*melody.Session)
			args[`from`] = GetAddrIP(s.GetWSConn().UnderlyingConn().RemoteAddr())
			if deviceConn, ok := args[`deviceConn`]; ok {
				delete(args, `deviceConn`)
				connUUID = deviceConn.(*melody.Session).UUID
				targetInfo = true
			}
		}
		if targetInfo {
			val, ok := Devices.Get(connUUID)
			if ok {
				device := val.(*modules.Device)
				args[`target`] = map[string]any{
					`name`: device.Hostname,
					`ip`:   device.WAN,
				}
			}
		}
	}
	output, _ := utils.JSON.MarshalToString(args)
	return output
}

func Info(ctx any, event, status, msg string, args map[string]any) {
	golog.Infof(getLog(ctx, event, status, msg, args))
}

func Warn(ctx any, event, status, msg string, args map[string]any) {
	golog.Warnf(getLog(ctx, event, status, msg, args))
}

func Error(ctx any, event, status, msg string, args map[string]any) {
	golog.Error(getLog(ctx, event, status, msg, args))
}

func Fatal(ctx any, event, status, msg string, args map[string]any) {
	golog.Fatalf(getLog(ctx, event, status, msg, args))
}

func Debug(ctx any, event, status, msg string, args map[string]any) {
	golog.Debugf(getLog(ctx, event, status, msg, args))
}

func CloseLog() {
	disposed = true
	golog.SetOutput(os.Stdout)
	if logWriter != nil {
		logWriter.Close()
		logWriter = nil
	}
}
