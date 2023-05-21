package terminal

import (
	"errors"
)

var (
	errDataNotFound = errors.New(`no input found in packet`)
	errDataInvalid  = errors.New(`can not parse data in packet`)
	errUUIDNotFound = errors.New(`can not find terminal identifier`)
)

// packet explanation:

// +---------+---------+----------+-------------+------+
// | magic   | op code | event id | data length | data |
// +---------+---------+----------+-------------+------+
// | 5 bytes | 1 byte  | 16 bytes | 2 bytes     | -    |
// +---------+---------+----------+-------------+------+

// magic:
// []byte{34, 22, 19, 17, 21}

// op code:
// 00: binary packet
// 01: JSON packet
