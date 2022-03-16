package config

type Cfg struct {
	Listen  string            `json:"listen"`
	Salt    string            `json:"salt"`
	Auth    map[string]string `json:"auth"`
	StdSalt []byte            `json:"-"`
}

var Config Cfg
