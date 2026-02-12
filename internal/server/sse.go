package server

import (
	"bufio"
	"clicktrainer/internal/events"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

type HxEventMessage struct {
	Event string
	Msg   string
}

var (
	clients   = make(map[chan HxEventMessage]bool)
	clientsMu sync.Mutex
)

func init() {
	go func() {
		for ev := range events.SceneChanges {
			BroadcastOOB("sceneChange", ev.Scene)
		}
	}()
}

func handleEvents(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[Handle:Events] Request Received")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	msgChan := make(chan HxEventMessage)
	clientsMu.Lock()
	clients[msgChan] = true
	clientsMu.Unlock()

	defer func() {
		clientsMu.Lock()
		delete(clients, msgChan)
		clientsMu.Unlock()
		close(msgChan)
	}()

	for {
		select {
		case <-r.Context().Done():
			clientsMu.Lock()
			defer clientsMu.Unlock()
			delete(clients, msgChan)
			return
		case msg := <-msgChan:
			scanner := bufio.NewScanner(strings.NewReader(msg.Msg))
			fmt.Println("[SSE] sending event", msg.Event)

			fmt.Fprintf(w, "event: %s\n", msg.Event)
			for scanner.Scan() {
				fmt.Fprintf(w, "data: %v\n", scanner.Text())
			}

			fmt.Fprint(w, "\n\n")
			flusher.Flush()
		}
	}

}

func BroadcastGame() {
	message := HxEventMessage{Event: "Update", Msg: "ping"}
	clientsMu.Lock()
	defer clientsMu.Unlock()
	for ch := range clients {
		select {
		case ch <- message:
		default:
			// skip clients with full data channels
		}
	}
}

func BroadcastOOB(event string, message string) {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	for ch := range clients {
		select {
		case ch <- HxEventMessage{Event: event, Msg: message}:
		default:
			// skip clients with full data channels
		}
	}
}
