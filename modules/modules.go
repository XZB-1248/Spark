package modules

type Packet struct {
	Code int                    `json:"code"`
	Act  string                 `json:"act,omitempty"`
	Msg  string                 `json:"msg,omitempty"`
	Data map[string]interface{} `json:"data,omitempty"`
}

type CommonPack struct {
	Code int         `json:"code"`
	Act  string      `json:"act,omitempty"`
	Msg  string      `json:"msg,omitempty"`
	Data interface{} `json:"data,omitempty"`
}

type Device struct {
	ID       string `json:"id"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	CPU      string `json:"cpu"`
	LAN      string `json:"lan"`
	WAN      string `json:"wan"`
	Mac      string `json:"mac"`
	Mem      uint64 `json:"mem"`
	Uptime   uint64 `json:"uptime"`
	Hostname string `json:"hostname"`
	Username string `json:"username"`
}
