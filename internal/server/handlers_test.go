package server

import (
	"bufio"
	"clicktrainer/internal/gamedata"
	"clicktrainer/internal/rooms"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/coder/websocket"
)

func newTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	cfg := gamedata.Config{
		RoundDuration:  5,
		InitialTargets: 2,
		CountdownSecs:  1,
	}
	roomStore := rooms.NewStore(cfg)

	funcMap := template.FuncMap{
		"inc": func(i int) int { return i + 1 },
	}
	tmpl := template.Must(template.New("").Funcs(funcMap).ParseFiles(
		"../../templates/home.html",
		"../../templates/game.html",
		"../../templates/join.html",
		"../../templates/target.html",
		"../../templates/lobby.html",
		"../../templates/recap.html",
		"../../templates/analytics/dashboard.html",
		"../../templates/analytics/leaderboard.html",
		"../../templates/analytics/player.html",
		"../../templates/analytics/game.html",
	))

	srv := &Server{
		Rooms: roomStore,
		Tmpl:  tmpl,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", srv.handleHome)
	mux.HandleFunc("/rooms/create", srv.handleCreateRoom)
	mux.HandleFunc("/rooms/join", srv.handleJoinRoom)
	mux.HandleFunc("GET /room/{code}", srv.handleRoomWithCode)
	mux.HandleFunc("GET /room", srv.handleRoom)
	mux.HandleFunc("POST /room/register", srv.handleRegister)
	mux.HandleFunc("POST /room/ready", srv.handleReady)
	mux.HandleFunc("POST /room/target/", srv.handleTarget)
	mux.HandleFunc("GET /room/ws", srv.handleWebSocket)
	mux.HandleFunc("POST /room/leave", srv.handleLeaveRoom)
	mux.HandleFunc("GET /room/events", srv.handleEvents)
	mux.HandleFunc("GET /room/poll", srv.handlePoll)
	mux.HandleFunc("POST /room/play-again", srv.handlePlayAgain)
	mux.HandleFunc("/health", srv.handleHealth)
	mux.HandleFunc("/analytics", srv.handleAnalyticsDashboard)
	mux.HandleFunc("/analytics/leaderboard", srv.handleAnalyticsLeaderboard)
	mux.HandleFunc("/analytics/player/", srv.handleAnalyticsPlayer)
	mux.HandleFunc("/analytics/game/", srv.handleAnalyticsGame)

	ts := httptest.NewServer(mux)
	return srv, ts
}

func newClientWithJar(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// createRoomAndGetCode creates a room via the API and returns the room code from the cookie.
func createRoomAndGetCode(t *testing.T, client *http.Client, baseURL string) string {
	t.Helper()
	resp, err := client.PostForm(baseURL+"/rooms/create", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	u, _ := url.Parse(baseURL)
	for _, c := range client.Jar.Cookies(u) {
		if c.Name == "room_code" {
			return c.Value
		}
	}
	t.Fatal("room_code cookie not set after create")
	return ""
}

func TestHandleHome(t *testing.T) {
	_, ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestHandleCreateRoom(t *testing.T) {
	_, ts := newTestServer(t)
	defer ts.Close()

	client := newClientWithJar(t)
	code := createRoomAndGetCode(t, client, ts.URL)

	if len(code) != 4 {
		t.Errorf("room code length = %d, want 4", len(code))
	}
}

func TestHandleJoinRoom_Valid(t *testing.T) {
	srv, ts := newTestServer(t)
	defer ts.Close()

	// Create a room directly
	room, _ := srv.Rooms.Create("host")

	client := newClientWithJar(t)
	resp, err := client.PostForm(ts.URL+"/rooms/join", url.Values{
		"code": {room.Code},
	})
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusSeeOther {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusSeeOther)
	}

	// Verify cookie
	u, _ := url.Parse(ts.URL)
	found := false
	for _, c := range client.Jar.Cookies(u) {
		if c.Name == "room_code" && c.Value == room.Code {
			found = true
		}
	}
	if !found {
		t.Error("room_code cookie not set after join")
	}
}

func TestHandleJoinRoom_Invalid(t *testing.T) {
	_, ts := newTestServer(t)
	defer ts.Close()

	client := newClientWithJar(t)
	resp, err := client.PostForm(ts.URL+"/rooms/join", url.Values{
		"code": {"ZZZZ"},
	})
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	// Should render home page with error, not redirect
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestHandleRegister_InRoom(t *testing.T) {
	srv, ts := newTestServer(t)
	defer ts.Close()

	room, _ := srv.Rooms.Create("host")

	client := newClientWithJar(t)
	// Set room cookie
	u, _ := url.Parse(ts.URL)
	client.Jar.SetCookies(u, []*http.Cookie{
		{Name: "room_code", Value: room.Code},
	})

	resp, err := client.PostForm(ts.URL+"/room/register", url.Values{
		"name": {"Alice"},
	})
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusSeeOther {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusSeeOther)
	}

	// Check player was added to the room
	players := room.Game.Players.GetList()
	if len(players) != 1 {
		t.Fatalf("expected 1 player, got %d", len(players))
	}
	if players[0].Name != "Alice" {
		t.Errorf("player name = %q, want %q", players[0].Name, "Alice")
	}
}

func TestHandleReady_InRoom(t *testing.T) {
	srv, ts := newTestServer(t)
	defer ts.Close()

	room, _ := srv.Rooms.Create("host")
	room.Game.Players.Add("test-id", "Alice")

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, _ := http.NewRequest("POST", ts.URL+"/room/ready", strings.NewReader("ready=ready"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "room_code", Value: room.Code})
	req.AddCookie(&http.Cookie{Name: "player_id", Value: "test-id"})

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	p := room.Game.Players.Get("test-id")
	if !p.Ready {
		t.Error("player should be ready after POST /room/ready")
	}
}

func TestHandleTarget_InRoom(t *testing.T) {
	srv, ts := newTestServer(t)
	defer ts.Close()

	room, _ := srv.Rooms.Create("host")
	room.Game.Players.Add("test-id", "Alice")
	target := room.Game.Targets.Add()

	client := &http.Client{}
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/room/target/%d/3", ts.URL, target.ID), nil)
	req.AddCookie(&http.Cookie{Name: "room_code", Value: room.Code})
	req.AddCookie(&http.Cookie{Name: "player_id", Value: "test-id"})

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	p := room.Game.Players.Get("test-id")
	if p.Score != 3 {
		t.Errorf("score = %d, want 3", p.Score)
	}
}

func TestHandlePlayAgain_InRoom(t *testing.T) {
	srv, ts := newTestServer(t)
	defer ts.Close()

	room, _ := srv.Rooms.Create("host")
	room.Game.Players.Add("test-id", "Alice")
	room.Game.Players.UpdateScore("test-id", 100)

	// Drain scene change events
	go func() { <-room.Game.Events.SceneChanges }()
	room.Game.SetScene(gamedata.SceneRecap)
	go func() { <-room.Game.Events.SceneChanges }()

	client := &http.Client{}
	req, _ := http.NewRequest("POST", ts.URL+"/room/play-again", nil)
	req.AddCookie(&http.Cookie{Name: "room_code", Value: room.Code})
	req.AddCookie(&http.Cookie{Name: "player_id", Value: "test-id"})

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if room.Game.Scene() != gamedata.SceneLobby {
		t.Errorf("scene = %q, want %q", room.Game.Scene(), gamedata.SceneLobby)
	}

	p := room.Game.Players.Get("test-id")
	if p.Score != 0 {
		t.Errorf("score = %d, want 0", p.Score)
	}
}

func TestHandleHealth_NoDB(t *testing.T) {
	_, ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body, _ := io.ReadAll(resp.Body)
	expected := `{"status":"ok"}`
	if string(body) != expected {
		t.Errorf("body = %q, want %q", string(body), expected)
	}
}

func TestHandleAnalytics_NoDB(t *testing.T) {
	endpoints := []string{
		"/analytics",
		"/analytics/leaderboard",
		"/analytics/player/someid",
		"/analytics/game/someid",
	}

	_, ts := newTestServer(t)
	defer ts.Close()

	for _, ep := range endpoints {
		t.Run(ep, func(t *testing.T) {
			resp, err := http.Get(ts.URL + ep)
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusServiceUnavailable {
				t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
			}
		})
	}
}

func TestHandleEvents_SSEStream(t *testing.T) {
	srv, ts := newTestServer(t)
	defer ts.Close()

	room, _ := srv.Rooms.Create("host")
	room.Game.Players.Add("test-id", "Alice")

	req, _ := http.NewRequest("GET", ts.URL+"/room/events", nil)
	req.AddCookie(&http.Cookie{Name: "room_code", Value: room.Code})
	req.AddCookie(&http.Cookie{Name: "player_id", Value: "test-id"})

	// Make request in goroutine since client.Do blocks until headers are flushed
	// (which only happens after the first SSE message is written)
	type sseResult struct {
		resp *http.Response
		err  error
	}
	resultCh := make(chan sseResult, 1)
	client := &http.Client{}
	go func() {
		resp, err := client.Do(req)
		resultCh <- sseResult{resp, err}
	}()

	// Wait for the handler to subscribe to the broadcaster
	time.Sleep(100 * time.Millisecond)

	// Broadcast triggers the first write+flush, which sends headers
	room.Broadcaster.BroadcastOOB("testEvent", "hello")

	select {
	case res := <-resultCh:
		if res.err != nil {
			t.Fatal(res.err)
		}
		defer res.resp.Body.Close()

		if res.resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want %d", res.resp.StatusCode, http.StatusOK)
		}

		if ct := res.resp.Header.Get("Content-Type"); ct != "text/event-stream" {
			t.Errorf("Content-Type = %q, want %q", ct, "text/event-stream")
		}

		// Read SSE lines in a goroutine with a timeout
		var gotEvent, gotData bool
		done := make(chan struct{})
		go func() {
			defer close(done)
			scanner := bufio.NewScanner(res.resp.Body)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "event: testEvent") {
					gotEvent = true
				}
				if strings.HasPrefix(line, "data: hello") {
					gotData = true
				}
				if gotEvent && gotData {
					return
				}
			}
		}()

		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Fatal("timed out reading SSE stream")
		}

		if !gotEvent {
			t.Error("did not receive event: testEvent")
		}
		if !gotData {
			t.Error("did not receive data: hello")
		}

	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for SSE connection")
	}
}

func TestHandleEvents_NoRoom(t *testing.T) {
	_, ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/room/events")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestRoomIsolation_TwoRooms(t *testing.T) {
	srv, ts := newTestServer(t)
	defer ts.Close()

	room1, _ := srv.Rooms.Create("host-1")
	room2, _ := srv.Rooms.Create("host-2")

	// Add player to room1
	room1.Game.Players.Add("p1", "Alice")

	// Add player to room2
	room2.Game.Players.Add("p2", "Bob")

	// Verify isolation
	r1Players := room1.Game.Players.GetList()
	r2Players := room2.Game.Players.GetList()

	if len(r1Players) != 1 || r1Players[0].Name != "Alice" {
		t.Error("room1 should only contain Alice")
	}
	if len(r2Players) != 1 || r2Players[0].Name != "Bob" {
		t.Error("room2 should only contain Bob")
	}
}

func TestHandleRoomWithCode_DeepLink(t *testing.T) {
	srv, ts := newTestServer(t)
	defer ts.Close()

	room, _ := srv.Rooms.Create("host")

	client := newClientWithJar(t)

	// Access deep link without player cookie — should render join page
	resp, err := client.Get(ts.URL + "/room/" + room.Code)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), room.Code) {
		t.Error("deep link page should contain the room code")
	}

	// Verify room_code cookie was set
	u, _ := url.Parse(ts.URL)
	found := false
	for _, c := range client.Jar.Cookies(u) {
		if c.Name == "room_code" && c.Value == room.Code {
			found = true
		}
	}
	if !found {
		t.Error("room_code cookie not set after deep link access")
	}
}

func TestHandleRoomWithCode_InvalidCode(t *testing.T) {
	_, ts := newTestServer(t)
	defer ts.Close()

	client := newClientWithJar(t)
	resp, err := client.Get(ts.URL + "/room/ZZZZ")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	// Should redirect to home
	if resp.StatusCode != http.StatusSeeOther {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusSeeOther)
	}
	loc := resp.Header.Get("Location")
	if loc != "/" {
		t.Errorf("location = %q, want %q", loc, "/")
	}
}

func TestHandleLeaveRoom_InLobby(t *testing.T) {
	srv, ts := newTestServer(t)
	defer ts.Close()

	room, _ := srv.Rooms.Create("host")
	room.Game.Players.Add("p1", "Alice")
	room.Game.Players.Add("p2", "Bob")

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, _ := http.NewRequest("POST", ts.URL+"/room/leave", nil)
	req.AddCookie(&http.Cookie{Name: "room_code", Value: room.Code})
	req.AddCookie(&http.Cookie{Name: "player_id", Value: "p1"})
	req.AddCookie(&http.Cookie{Name: "player_name", Value: "Alice"})

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	// Should have HX-Redirect header
	if loc := resp.Header.Get("HX-Redirect"); loc != "/" {
		t.Errorf("HX-Redirect = %q, want %q", loc, "/")
	}

	// Player should be removed
	if room.Game.Players.Get("p1") != nil {
		t.Error("player p1 should be removed from room")
	}

	// Room should still exist (Bob is still in it)
	if srv.Rooms.Get(room.Code) == nil {
		t.Error("room should still exist with remaining player")
	}
	if room.Game.Players.Count() != 1 {
		t.Errorf("player count = %d, want 1", room.Game.Players.Count())
	}
}

func TestHandleLeaveRoom_LastPlayer(t *testing.T) {
	srv, ts := newTestServer(t)
	defer ts.Close()

	room, _ := srv.Rooms.Create("host")
	room.Game.Players.Add("p1", "Alice")

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, _ := http.NewRequest("POST", ts.URL+"/room/leave", nil)
	req.AddCookie(&http.Cookie{Name: "room_code", Value: room.Code})
	req.AddCookie(&http.Cookie{Name: "player_id", Value: "p1"})

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	// Room should be deleted
	if srv.Rooms.Get(room.Code) != nil {
		t.Error("room should be deleted when last player leaves")
	}
}

func TestHandleWebSocket_Click(t *testing.T) {
	srv, ts := newTestServer(t)
	defer ts.Close()

	room, _ := srv.Rooms.Create("host")
	room.Game.Players.Add("ws-player", "Alice")
	target := room.Game.Targets.Add()

	// Build WebSocket URL
	wsURL := strings.Replace(ts.URL, "http://", "ws://", 1) + "/room/ws"

	// Use nhooyr.io/websocket client
	header := http.Header{}
	header.Set("Cookie", fmt.Sprintf("room_code=%s; player_id=ws-player", room.Code))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		HTTPHeader: header,
	})
	if err != nil {
		t.Fatalf("WebSocket dial error: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Send click message (new format: t=click, id=targetID, p=points)
	msg := fmt.Sprintf(`{"t":"click","id":%d,"p":3}`, target.ID)
	if err := conn.Write(ctx, websocket.MessageText, []byte(msg)); err != nil {
		t.Fatalf("WebSocket write error: %v", err)
	}

	// Give processClick time to execute
	time.Sleep(100 * time.Millisecond)

	p := room.Game.Players.Get("ws-player")
	if p.Score != 3 {
		t.Errorf("score = %d, want 3", p.Score)
	}
}

func TestProcessClick_InvalidPoints(t *testing.T) {
	srv, _ := newTestServer(t)

	room, _ := srv.Rooms.Create("host")
	room.Game.Players.Add("p1", "Alice")
	target := room.Game.Targets.Add()

	// Points = 0 should be rejected
	srv.processClick(room, "p1", target.ID, 0)
	p := room.Game.Players.Get("p1")
	if p.Score != 0 {
		t.Errorf("score = %d, want 0 (invalid points should be rejected)", p.Score)
	}

	// Points = 5 should be rejected
	target2 := room.Game.Targets.Add()
	srv.processClick(room, "p1", target2.ID, 5)
	p = room.Game.Players.Get("p1")
	if p.Score != 0 {
		t.Errorf("score = %d, want 0 (invalid points should be rejected)", p.Score)
	}
}

func TestHandleRoom_RedirectsToDeepLink(t *testing.T) {
	srv, ts := newTestServer(t)
	defer ts.Close()

	room, _ := srv.Rooms.Create("host")

	client := newClientWithJar(t)
	u, _ := url.Parse(ts.URL)
	client.Jar.SetCookies(u, []*http.Cookie{
		{Name: "room_code", Value: room.Code},
	})

	resp, err := client.Get(ts.URL + "/room")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusSeeOther {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusSeeOther)
	}
	loc := resp.Header.Get("Location")
	expected := "/room/" + room.Code
	if loc != expected {
		t.Errorf("location = %q, want %q", loc, expected)
	}
}
