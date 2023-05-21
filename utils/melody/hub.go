package melody

import (
	"Spark/utils/cmap"
)

type hub struct {
	sessions   cmap.ConcurrentMap[string, *Session]
	queue      chan *envelope
	register   chan *Session
	unregister chan *Session
	exit       chan *envelope
	open       bool
}

func newHub() *hub {
	return &hub{
		sessions:   cmap.New[*Session](),
		queue:      make(chan *envelope),
		register:   make(chan *Session),
		unregister: make(chan *Session),
		exit:       make(chan *envelope),
		open:       true,
	}
}

func (h *hub) run() {
loop:
	for {
		select {
		case s := <-h.register:
			if h.open {
				h.sessions.Set(s.UUID, s)
			}
		case s := <-h.unregister:
			h.sessions.Remove(s.UUID)
		case m := <-h.queue:
			if len(m.list) > 0 {
				for _, uuid := range m.list {
					if s, ok := h.sessions.Get(uuid); ok {
						s.writeMessage(m)
					}
				}
			} else if m.filter == nil {
				h.sessions.IterCb(func(uuid string, s *Session) bool {
					s.writeMessage(m)
					return true
				})
			} else {
				h.sessions.IterCb(func(uuid string, s *Session) bool {
					if m.filter(s) {
						s.writeMessage(m)
					}
					return true
				})
			}
		case m := <-h.exit:
			var keys []string
			h.open = false
			h.sessions.IterCb(func(uuid string, s *Session) bool {
				s.writeMessage(m)
				s.Close()
				keys = append(keys, uuid)
				return true
			})
			for i := range keys {
				h.sessions.Remove(keys[i])
			}
			break loop
		}
	}
}

func (h *hub) closed() bool {
	return !h.open
}

func (h *hub) len() int {
	return h.sessions.Count()
}

func (h *hub) list() []string {
	return h.sessions.Keys()
}
