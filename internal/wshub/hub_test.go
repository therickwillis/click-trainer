package wshub

import (
	"encoding/json"
	"testing"
	"time"
)

func TestRegisterAndBroadcast(t *testing.T) {
	h := NewHub()

	c1 := &Client{PlayerID: "p1", Name: "Alice", Color: "#ff0000", Send: make(chan []byte, 16)}
	c2 := &Client{PlayerID: "p2", Name: "Bob", Color: "#00ff00", Send: make(chan []byte, 16)}
	c3 := &Client{PlayerID: "p3", Name: "Carol", Color: "#0000ff", Send: make(chan []byte, 16)}

	h.Register(c1)
	h.Register(c2)
	h.Register(c3)

	msg := ServerMessage{Type: "move", PlayerID: "p1", X: 100, Y: 200}
	h.BroadcastExcept("p1", msg)

	// c2 and c3 should receive the message, c1 should not
	select {
	case data := <-c2.Send:
		var got ServerMessage
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if got.Type != "move" || got.X != 100 || got.Y != 200 {
			t.Fatalf("unexpected message: %+v", got)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("c2 did not receive message")
	}

	select {
	case <-c3.Send:
		// expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("c3 did not receive message")
	}

	select {
	case <-c1.Send:
		t.Fatal("c1 should not receive its own message")
	default:
		// expected
	}
}

func TestUnregisterBroadcastsLeave(t *testing.T) {
	h := NewHub()

	c1 := &Client{PlayerID: "p1", Name: "Alice", Send: make(chan []byte, 16)}
	c2 := &Client{PlayerID: "p2", Name: "Bob", Send: make(chan []byte, 16)}

	h.Register(c1)
	h.Register(c2)

	h.Unregister("p1")

	// c2 should receive a leave message
	select {
	case data := <-c2.Send:
		var got ServerMessage
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if got.Type != "leave" || got.PlayerID != "p1" {
			t.Fatalf("expected leave for p1, got: %+v", got)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("c2 did not receive leave message")
	}

	// c1's Send channel should be closed
	_, ok := <-c1.Send
	if ok {
		t.Fatal("c1.Send should be closed")
	}
}

func TestUnregisterNonexistent(t *testing.T) {
	h := NewHub()
	// Should not panic
	h.Unregister("nonexistent")
}

func TestBroadcastDropsWhenFull(t *testing.T) {
	h := NewHub()

	// Channel with capacity 1
	c := &Client{PlayerID: "p1", Send: make(chan []byte, 1)}
	h.Register(c)

	// Fill the channel
	c.Send <- []byte("filler")

	// This should not block — message dropped
	h.BroadcastExcept("other", ServerMessage{Type: "move", X: 1, Y: 2})

	// Only the filler should be in the channel
	data := <-c.Send
	if string(data) != "filler" {
		t.Fatalf("expected filler, got: %s", data)
	}

	select {
	case <-c.Send:
		t.Fatal("should be empty after draining filler")
	default:
		// expected
	}
}
