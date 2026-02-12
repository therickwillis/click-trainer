package server

import (
	"clicktrainer/internal/gamedata"
	"clicktrainer/internal/rooms"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"text/template"
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
	mux.HandleFunc("/room", srv.handleRoom)
	mux.HandleFunc("/room/register", srv.handleRegister)
	mux.HandleFunc("/room/ready", srv.handleReady)
	mux.HandleFunc("/room/target/", srv.handleTarget)
	mux.HandleFunc("/room/events", srv.handleEvents)
	mux.HandleFunc("/room/poll", srv.handlePoll)
	mux.HandleFunc("/room/play-again", srv.handlePlayAgain)

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
