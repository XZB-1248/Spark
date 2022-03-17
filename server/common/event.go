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

// CallEvent 负责判断packet中的Callback字段，如果存在该字段，
// 就会调用event中的函数，并在调用完成之后通过chan通知addOnceEvent调用方
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

// AddEventOnce 会添加一个一次性的回调命令，client可以对事件成功与否进行回复
// trigger一般是uuid，以此尽可能保证事件的独一无二
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

// AddEvent 会添加一个持续的回调命令，client可以对事件成功与否进行回复
// trigger一般是uuid，以此尽可能保证事件的独一无二
func AddEvent(fn EventCallback, connUUID, trigger string) {
	ev := &event{
		connection: connUUID,
		callback:   fn,
		channel:    nil,
	}
	eventTable.Set(trigger, ev)
}

// RemoveEvent 会删除特定的回调命令
func RemoveEvent(trigger string) {
	eventTable.Remove(trigger)
}

// HasEvent returns if the event exists.
func HasEvent(trigger string) bool {
	return eventTable.Has(trigger)
}
