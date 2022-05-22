package common

import "time"

var Unix int64 = time.Now().Unix()

// To prevent call time.Now().Unix() too often.
func init() {
	go func() {
		for now := range time.NewTicker(time.Second).C {
			Unix = now.Unix()
		}
	}()
}
