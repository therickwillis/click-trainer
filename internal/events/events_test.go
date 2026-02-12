package events

import (
	"testing"
	"time"
)

func TestNewBus(t *testing.T) {
	bus := NewBus()
	if bus == nil {
		t.Fatal("NewBus() returned nil")
	}
	if bus.SceneChanges == nil {
		t.Fatal("SceneChanges channel is nil")
	}
}

func TestBus_SendReceive(t *testing.T) {
	bus := NewBus()
	ev := SceneChangeEvent{Scene: "combat"}

	go func() {
		bus.SceneChanges <- ev
	}()

	select {
	case received := <-bus.SceneChanges:
		if received.Scene != "combat" {
			t.Errorf("received Scene = %q, want %q", received.Scene, "combat")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestBus_Buffered(t *testing.T) {
	bus := NewBus()

	// Should be able to send up to 10 without blocking
	for i := 0; i < 10; i++ {
		bus.SceneChanges <- SceneChangeEvent{Scene: "test"}
	}

	// Drain
	for i := 0; i < 10; i++ {
		<-bus.SceneChanges
	}
}
