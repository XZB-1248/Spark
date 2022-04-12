package common

import (
	"Spark/modules"
	"Spark/utils/cmap"
	"Spark/utils/melody"
	"time"
)

type event struct {
	connection string
	callback   EventCallback
	channel    chan bool
}
type EventCallback func(modules.Packet, *melody.Session)

var eventTable = cmap.New()

// CallEvent tries to call the callback with the given uuid
// after that, it will notify the caller via the channel
func CallEvent(pack modules.Packet, session *melody.Session) {
	if len(pack.Event) == 0 {
		return
	}
	v, ok := eventTable.Get(pack.Event)
	if !ok {
		return
	}
	ev := v.(*event)
	if session != nil && session.UUID != ev.connection {
		return
	}
	ev.callback(pack, session)
	if ev.channel != nil {
		defer close(ev.channel)
		select {
		case ev.channel <- true:
		default:
		}
	}
}

// AddEventOnce adds a new event only once and client
// can call back the event with the given event trigger.
// Event trigger should be uuid to make every event unique.
func AddEventOnce(fn EventCallback, connUUID, trigger string, timeout time.Duration) bool {
	done := make(chan bool)
	ev := &event{
		connection: connUUID,
		callback:   fn,
		channel:    done,
	}
	eventTable.Set(trigger, ev)
	defer eventTable.Remove(trigger)
	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

// AddEvent adds a new event and client can call back
// the event with the given event trigger.
func AddEvent(fn EventCallback, connUUID, trigger string) {
	ev := &event{
		connection: connUUID,
		callback:   fn,
		channel:    nil,
	}
	eventTable.Set(trigger, ev)
}

// RemoveEvent deletes the event with the given event trigger.
func RemoveEvent(trigger string) {
	eventTable.Remove(trigger)
}

// HasEvent returns if the event exists.
func HasEvent(trigger string) bool {
	return eventTable.Has(trigger)
}
