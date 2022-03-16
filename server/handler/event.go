package handler

import (
	"Spark/modules"
	"Spark/utils/cmap"
	"Spark/utils/melody"
	"time"
)

type event struct {
	connection string
	callback   eventCb
	channel    chan bool
}
type eventCb func(modules.Packet, *melody.Session)

var eventTable = cmap.New()

// evCaller 负责判断packet中的Callback字段，如果存在该字段，
// 就会调用event中的函数，并在调用完成之后通过chan通知addOnceEvent调用方
func evCaller(pack modules.Packet, session *melody.Session) {
	if pack.Data == nil {
		return
	}
	v, ok := pack.Data[`callback`]
	if !ok {
		return
	}
	trigger, ok := v.(string)
	if !ok {
		return
	}
	v, ok = eventTable.Get(trigger)
	if !ok {
		return
	}
	ev := v.(*event)
	if session != nil && session.UUID != ev.connection {
		return
	}
	delete(pack.Data, `callback`)
	ev.callback(pack, session)
	if ev.channel != nil {
		defer close(ev.channel)
		select {
		case ev.channel <- true:
		default:
		}
	}
}

// addEventOnce 会添加一个一次性的回调命令，client可以对事件成功与否进行回复
// trigger一般是uuid，以此尽可能保证事件的独一无二
func addEventOnce(fn eventCb, connUUID, trigger string, timeout time.Duration) bool {
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

// addEvent 会添加一个持续的回调命令，client可以对事件成功与否进行回复
// trigger一般是uuid，以此尽可能保证事件的独一无二
func addEvent(fn eventCb, connUUID, trigger string) {
	ev := &event{
		connection: connUUID,
		callback:   fn,
		channel:    nil,
	}
	eventTable.Set(trigger, ev)
}

// removeEvent 会删除特定的回调命令
func removeEvent(trigger string) {
	eventTable.Remove(trigger)
}

// hasEvent returns if the event exists.
func hasEvent(trigger string) bool {
	return eventTable.Has(trigger)
}
