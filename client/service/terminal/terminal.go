package terminal

import (
	"Spark/utils/cmap"
	"errors"
)

var terminals = cmap.New()
var (
	errDataNotFound = errors.New(`no input found in packet`)
	errDataInvalid  = errors.New(`can not parse data in packet`)
	errUUIDNotFound = errors.New(`can not find terminal identifier`)
)