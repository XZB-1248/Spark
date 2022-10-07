package config

import (
	"Spark/utils"
	"bytes"
	"flag"
	"github.com/kataras/golog"
	"os"
)

type config struct {
	Listen    string            `json:"listen"`
	Salt      string            `json:"salt"`
	Auth      map[string]string `json:"auth"`
	Log       *log              `json:"log"`
	SaltBytes []byte            `json:"-"`
}
type log struct {
	Level string `json:"level"`
	Path  string `json:"path"`
	Days  uint   `json:"days"`
}

// COMMIT is hash of this commit, for auto upgrade.
var COMMIT = ``
var Config config
var BuiltPath = `./built/%v_%v`

func init() {
	golog.SetTimeFormat(`2006/01/02 15:04:05`)

	var (
		err                      error
		configData               []byte
		configPath, listen, salt string
		username, password       string
		logLevel, logPath        string
		logDays                  uint
	)
	flag.StringVar(&configPath, `config`, `config.json`, `config file path, default: config.json`)
	flag.StringVar(&listen, `listen`, `:8000`, `required, listen address, default: :8000`)
	flag.StringVar(&salt, `salt`, ``, `required, salt of server`)
	flag.StringVar(&username, `username`, ``, `username of web interface`)
	flag.StringVar(&password, `password`, ``, `password of web interface`)
	flag.StringVar(&logLevel, `log-level`, `info`, `log level, default: info`)
	flag.StringVar(&logPath, `log-path`, `./logs`, `log file path, default: ./logs`)
	flag.UintVar(&logDays, `log-days`, 7, `max days of logs, default: 7`)
	flag.Parse()

	if len(configPath) > 0 {
		configData, err = os.ReadFile(configPath)
		if err != nil {
			configData, err = os.ReadFile(`Config.json`)
			if err != nil {
				fatal(map[string]any{
					`event`:  `CONFIG_LOAD`,
					`status`: `fail`,
					`msg`:    err.Error(),
				})
				return
			}
		}
		err = utils.JSON.Unmarshal(configData, &Config)
		if err != nil {
			fatal(map[string]any{
				`event`:  `CONFIG_PARSE`,
				`status`: `fail`,
				`msg`:    err.Error(),
			})
			return
		}
		if Config.Log == nil {
			Config.Log = &log{
				Level: `info`,
				Path:  `./logs`,
				Days:  7,
			}
		}
	} else {
		Config = config{
			Listen: listen,
			Salt:   salt,
			Auth: map[string]string{
				username: password,
			},
			Log: &log{
				Level: logLevel,
				Path:  logPath,
				Days:  logDays,
			},
		}
	}

	if len(Config.Salt) > 24 {
		fatal(map[string]any{
			`event`:  `CONFIG_PARSE`,
			`status`: `fail`,
			`msg`:    `length of salt should less than 24`,
		})
		return
	}
	Config.SaltBytes = []byte(Config.Salt)
	Config.SaltBytes = append(Config.SaltBytes, bytes.Repeat([]byte{25}, 24)...)
	Config.SaltBytes = Config.SaltBytes[:24]

	golog.SetLevel(utils.If(len(Config.Log.Level) == 0, `info`, Config.Log.Level))
}

func fatal(args map[string]any) {
	output, _ := utils.JSON.MarshalToString(args)
	golog.Fatal(output)
}
