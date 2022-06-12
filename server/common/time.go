package common

import "time"

var Now time.Time = time.Now()
var Unix int64 = Now.Unix()

// To prevent call time.Now().Unix() too often.
func init() {
	go func() {
		for now := range time.NewTicker(time.Second).C {
			Now = now
			Unix = now.Unix()
		}
	}()
}
