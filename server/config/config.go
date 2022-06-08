package config

import (
	"os"
	"path/filepath"
)

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
var BuiltPath = getBuiltPath()

// COMMIT is hash of this commit, for auto upgrade.
var COMMIT = ``

func getBuiltPath() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return `./built/%v_%v`
	}
	return filepath.Join(dir, `built/%v_%v`)
}
