package modules

import "reflect"

type Packet struct {
	Code  int                    `json:"code"`
	Act   string                 `json:"act,omitempty"`
	Msg   string                 `json:"msg,omitempty"`
	Data  map[string]interface{} `json:"data,omitempty"`
	Event string                 `json:"event,omitempty"`
}

type CommonPack struct {
	Code  int         `json:"code"`
	Act   string      `json:"act,omitempty"`
	Msg   string      `json:"msg,omitempty"`
	Data  interface{} `json:"data,omitempty"`
	Event string      `json:"event,omitempty"`
}

type Device struct {
	ID       string `json:"id"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	LAN      string `json:"lan"`
	WAN      string `json:"wan"`
	MAC      string `json:"mac"`
	Net      Net    `json:"net"`
	CPU      CPU    `json:"cpu"`
	RAM      IO     `json:"ram"`
	Disk     IO     `json:"disk"`
	Uptime   uint64 `json:"uptime"`
	Latency  uint   `json:"latency"`
	Hostname string `json:"hostname"`
	Username string `json:"username"`
}

type IO struct {
	Total uint64  `json:"total"`
	Used  uint64  `json:"used"`
	Usage float64 `json:"usage"`
}

type CPU struct {
	Model string  `json:"model"`
	Usage float64 `json:"usage"`
	Cores struct {
		Logical  int `json:"logical"`
		Physical int `json:"physical"`
	} `json:"cores"`
}

type Net struct {
	Sent uint64 `json:"sent"`
	Recv uint64 `json:"recv"`
}

func (p *Packet) GetData(key string, t reflect.Kind) (interface{}, bool) {
	if p.Data == nil {
		return nil, false
	}
	data, ok := p.Data[key]
	if !ok {
		return nil, false
	}
	switch t {
	case reflect.String:
		val, ok := data.(string)
		return val, ok
	case reflect.Uint:
		val, ok := data.(uint)
		return val, ok
	case reflect.Uint32:
		val, ok := data.(uint32)
		return val, ok
	case reflect.Uint64:
		val, ok := data.(uint64)
		return val, ok
	case reflect.Int:
		val, ok := data.(int)
		return val, ok
	case reflect.Int64:
		val, ok := data.(int64)
		return val, ok
	case reflect.Bool:
		val, ok := data.(bool)
		return val, ok
	case reflect.Float64:
		val, ok := data.(float64)
		return val, ok
	default:
		return nil, false
	}
}
