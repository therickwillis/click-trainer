package broadcast

import (
	"clicktrainer/internal/events"
	"testing"
	"time"
)

func TestNewBroadcaster(t *testing.T) {
	bus := events.NewBus()
	b := NewBroadcaster(bus)
	if b == nil {
		t.Fatal("NewBroadcaster() returned nil")
	}
}

func TestBroadcaster_SubscribeUnsubscribe(t *testing.T) {
	bus := events.NewBus()
	b := NewBroadcaster(bus)

	ch := b.Subscribe()
	if ch == nil {
		t.Fatal("Subscribe() returned nil")
	}

	b.Mu.Lock()
	if len(b.Clients) != 1 {
		t.Errorf("clients count = %d, want 1", len(b.Clients))
	}
	b.Mu.Unlock()

	b.Unsubscribe(ch)

	b.Mu.Lock()
	if len(b.Clients) != 0 {
		t.Errorf("clients count after unsubscribe = %d, want 0", len(b.Clients))
	}
	b.Mu.Unlock()
}

func TestBroadcaster_BroadcastOOB(t *testing.T) {
	bus := events.NewBus()
	b := NewBroadcaster(bus)

	ch1 := b.Subscribe()
	ch2 := b.Subscribe()

	b.BroadcastOOB("test-event", "hello")

	select {
	case msg := <-ch1:
		if msg.Event != "test-event" || msg.Msg != "hello" {
			t.Errorf("ch1 got %+v, want event=test-event, msg=hello", msg)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("ch1 timed out")
	}

	select {
	case msg := <-ch2:
		if msg.Event != "test-event" || msg.Msg != "hello" {
			t.Errorf("ch2 got %+v, want event=test-event, msg=hello", msg)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("ch2 timed out")
	}

	b.Unsubscribe(ch1)
	b.Unsubscribe(ch2)
}

func TestBroadcaster_SkipsFullChannels(t *testing.T) {
	bus := events.NewBus()
	b := NewBroadcaster(bus)

	ch := b.Subscribe()

	// Fill the channel buffer (capacity 10)
	for i := 0; i < 10; i++ {
		b.BroadcastOOB("fill", "data")
	}

	// This should not block even though channel is full
	done := make(chan bool)
	go func() {
		b.BroadcastOOB("overflow", "data")
		done <- true
	}()

	select {
	case <-done:
		// Success - didn't block
	case <-time.After(1 * time.Second):
		t.Fatal("BroadcastOOB blocked on full channel")
	}

	b.Unsubscribe(ch)
}

func TestBroadcaster_SceneChangeForwarding(t *testing.T) {
	bus := events.NewBus()
	b := NewBroadcaster(bus)

	ch := b.Subscribe()

	bus.SceneChanges <- events.SceneChangeEvent{Scene: "combat"}

	select {
	case msg := <-ch:
		if msg.Event != "sceneChange" || msg.Msg != "combat" {
			t.Errorf("got %+v, want event=sceneChange, msg=combat", msg)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for scene change broadcast")
	}

	b.Unsubscribe(ch)
}
