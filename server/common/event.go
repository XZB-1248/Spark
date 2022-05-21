package common

import (
	"Spark/modules"
	"Spark/utils/cmap"
	"Spark/utils/melody"
	"time"
)

type EventCallback func(modules.Packet, *melody.Session)
type event struct {
	connection string
	callback   EventCallback
	finish     chan bool
	remove     chan bool
}

var events = cmap.New()

// CallEvent tries to call the callback with the given uuid
// after that, it will notify the caller via the channel
func CallEvent(pack modules.Packet, session *melody.Session) {
	if len(pack.Event) == 0 {
		return
	}
	v, ok := events.Get(pack.Event)
	if !ok {
		return
	}
	ev := v.(*event)
	if session != nil && session.UUID != ev.connection {
		return
	}
	ev.callback(pack, session)
	if ev.finish != nil {
		ev.finish <- true
	}
}

// AddEventOnce adds a new event only once and client
// can call back the event with the given event trigger.
// Event trigger should be uuid to make every event unique.
func AddEventOnce(fn EventCallback, connUUID, trigger string, timeout time.Duration) bool {
	ev := &event{
		connection: connUUID,
		callback:   fn,
		finish:     make(chan bool),
		remove:     make(chan bool),
	}
	events.Set(trigger, ev)
	defer events.Remove(trigger)
	defer close(ev.finish)
	defer close(ev.remove)
	select {
	case ok := <-ev.finish:
		return ok
	case ok := <-ev.remove:
		return ok
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
	}
	events.Set(trigger, ev)
}

// RemoveEvent deletes the event with the given event trigger.
// The ok will be returned to caller if the event is temp (only once).
func RemoveEvent(trigger string, ok ...bool) {
	v, found := events.Get(trigger)
	if !found {
		return
	}
	events.Remove(trigger)
	if ev := v.(*event); ev.remove != nil {
		if len(ok) > 0 {
			ev.remove <- ok[0]
		}
	}
}

// HasEvent returns if the event exists.
func HasEvent(trigger string) bool {
	return events.Has(trigger)
}
