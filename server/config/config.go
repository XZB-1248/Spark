package config

type Cfg struct {
	Debug struct {
		Pprof bool `json:"pprof"`
		Gin   bool `json:"gin"`
	} `json:"debug,omitempty"`
	Listen  string            `json:"listen"`
	Salt    string            `json:"salt"`
	Auth    map[string]string `json:"auth"`
	StdSalt []byte            `json:"-"`
}

var Config Cfg
var BuiltPath = `./built/%v_%v`

// COMMIT means this commit hash, for auto upgrade.
var COMMIT = ``
