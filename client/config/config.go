package config

import (
	"fmt"
	"net/url"
)

type Cfg struct {
	Secure bool   `json:"secure"`
	Host   string `json:"host"`
	Port   int    `json:"port"`
	Path   string `json:"path"`
	UUID   string `json:"uuid"`
	Key    string `json:"key"`
}

var Config Cfg

func GetBaseURL(ws bool) string {
	baseUrl := url.URL{
		Host: fmt.Sprintf(`%v:%v`, Config.Host, Config.Port),
		Path: Config.Path,
	}
	if ws {
		if Config.Secure {
			baseUrl.Scheme = `wss`
		} else {
			baseUrl.Scheme = `ws`
		}
	} else {
		if Config.Secure {
			baseUrl.Scheme = `https`
		} else {
			baseUrl.Scheme = `http`
		}
	}
	return baseUrl.String()
}
