package broadcast

import (
	"clicktrainer/internal/events"
	"sync"
)

type HxEventMessage struct {
	Event string
	Msg   string
}

type Broadcaster struct {
	Mu      sync.Mutex
	Clients map[chan HxEventMessage]bool
}

func NewBroadcaster(bus *events.Bus) *Broadcaster {
	b := &Broadcaster{
		Clients: make(map[chan HxEventMessage]bool),
	}
	go func() {
		for ev := range bus.SceneChanges {
			b.BroadcastOOB("sceneChange", ev.Scene)
		}
	}()
	return b
}

func (b *Broadcaster) Subscribe() chan HxEventMessage {
	ch := make(chan HxEventMessage, 10)
	b.Mu.Lock()
	b.Clients[ch] = true
	b.Mu.Unlock()
	return ch
}

func (b *Broadcaster) Unsubscribe(ch chan HxEventMessage) {
	b.Mu.Lock()
	delete(b.Clients, ch)
	b.Mu.Unlock()
	close(ch)
}

func (b *Broadcaster) BroadcastOOB(event string, message string) {
	b.Mu.Lock()
	defer b.Mu.Unlock()
	for ch := range b.Clients {
		select {
		case ch <- HxEventMessage{Event: event, Msg: message}:
		default:
			// skip clients with full data channels
		}
	}
}
