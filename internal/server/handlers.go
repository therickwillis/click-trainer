package server

import (
	"bytes"
	"clicktrainer/internal/analytics"
	"clicktrainer/internal/db"
	"clicktrainer/internal/gamedata"
	"clicktrainer/internal/rooms"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/google/uuid"
)

type Server struct {
	Rooms       *rooms.Store
	Tmpl        *template.Template
	DB          *db.DB          // nil if no database configured
	ClickBuffer chan db.ClickEvent // nil if no database configured
}

// getRoom resolves the current room from the room_code cookie.
func (s *Server) getRoom(r *http.Request) *rooms.Room {
	cookie, err := r.Cookie("room_code")
	if err != nil {
		return nil
	}
	return s.Rooms.Get(cookie.Value)
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	// If player has a room cookie and it's valid, redirect to /room/{code}
	if room := s.getRoom(r); room != nil {
		http.Redirect(w, r, "/room/"+room.Code, http.StatusSeeOther)
		return
	}
	if err := s.Tmpl.ExecuteTemplate(w, "home", nil); err != nil {
		log.Println(err)
		http.Error(w, "Error rendering home page", http.StatusInternalServerError)
	}
}

func (s *Server) handleCreateRoom(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[Handle:CreateRoom] Request Received")

	hostID := uuid.New().String()
	room, err := s.Rooms.Create(hostID)
	if err != nil {
		log.Println(err)
		http.Error(w, "Failed to create room", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "room_code",
		Value:    room.Code,
		Path:     "/",
		HttpOnly: true,
	})

	fmt.Printf("[Handle:CreateRoom] Created room %s\n", room.Code)
	http.Redirect(w, r, "/room/"+room.Code, http.StatusSeeOther)
}

func (s *Server) handleJoinRoom(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[Handle:JoinRoom] Request Received")

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	code := strings.ToUpper(strings.TrimSpace(r.FormValue("code")))
	room := s.Rooms.Get(code)
	if room == nil {
		// Re-render home with error
		if err := s.Tmpl.ExecuteTemplate(w, "home", map[string]string{"Error": "Room not found"}); err != nil {
			log.Println(err)
		}
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "room_code",
		Value:    code,
		Path:     "/",
		HttpOnly: true,
	})

	http.Redirect(w, r, "/room/"+code, http.StatusSeeOther)
}

// renderRoom contains the shared logic for rendering the game room view.
func (s *Server) renderRoom(w http.ResponseWriter, r *http.Request, room *rooms.Room) {
	idCookie, err := r.Cookie("player_id")
	if err == nil {
		nameCookie, err := r.Cookie("player_name")
		if err != nil {
			http.SetCookie(w, &http.Cookie{
				Name:   "player_id",
				MaxAge: -1,
			})
			http.Redirect(w, r, "/room/"+room.Code, http.StatusSeeOther)
			return
		}

		if !room.Game.Players.ValidateSession(idCookie.Value) {
			player := room.Game.Players.Add(idCookie.Value, nameCookie.Value)
			var buf bytes.Buffer
			if err := s.Tmpl.ExecuteTemplate(&buf, "lobbyPlayer", player); err != nil {
				log.Println(err)
			}
			room.Broadcaster.BroadcastOOB("newPlayer", buf.String())
		}

		data := room.Game.Get(idCookie.Value)
		data.RoomCode = room.Code

		if err := s.Tmpl.ExecuteTemplate(w, "game", data); err != nil {
			log.Println(err)
			http.Error(w, "Error rendering game view", http.StatusInternalServerError)
		}
		return
	}

	if err := s.Tmpl.ExecuteTemplate(w, "join", map[string]string{"RoomCode": room.Code}); err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleRoomWithCode(w http.ResponseWriter, r *http.Request) {
	code := strings.ToUpper(r.PathValue("code"))
	room := s.Rooms.Get(code)
	if room == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "room_code",
		Value:    room.Code,
		Path:     "/",
		HttpOnly: true,
	})

	s.renderRoom(w, r, room)
}

func (s *Server) handleRoom(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[Handle:Room] Request Received")
	room := s.getRoom(r)
	if room == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/room/"+room.Code, http.StatusSeeOther)
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[Handle:Register] Request Received")
	room := s.getRoom(r)
	if room == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	id := uuid.New().String()
	name := r.FormValue("name")
	fmt.Println("Registering name:", name)

	http.SetCookie(w, &http.Cookie{
		Name:     "player_id",
		Value:    id,
		Path:     "/",
		HttpOnly: true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "player_name",
		Value:    name,
		Path:     "/",
		HttpOnly: true,
	})

	player := room.Game.Players.Add(id, name)

	if s.DB != nil {
		if err := s.DB.UpsertPlayer(id, name, player.Color); err != nil {
			log.Printf("[DB] UpsertPlayer error: %v\n", err)
		}
	}

	data := room.Game.Get(id)

	switch data.Scene {
	case gamedata.SceneLobby:
		var buf bytes.Buffer
		if err := s.Tmpl.ExecuteTemplate(&buf, "lobbyPlayer", data.Player); err != nil {
			log.Println(err)
		}
		room.Broadcaster.BroadcastOOB("newPlayer", buf.String())
	default:
		var buf bytes.Buffer
		if err := s.Tmpl.ExecuteTemplate(&buf, "scoreboard", room.Game.Players.GetList()); err != nil {
			log.Println(err)
		}
		room.Broadcaster.BroadcastOOB("scoreboard", buf.String())
	}

	http.Redirect(w, r, "/room/"+room.Code, http.StatusSeeOther)
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[Handle:Ready] Request Received")
	room := s.getRoom(r)
	if room == nil {
		http.Error(w, "Room not found", http.StatusBadRequest)
		return
	}

	idCookie, err := r.Cookie("player_id")
	if err != nil {
		http.Error(w, "Not Registered", http.StatusBadRequest)
		return
	}

	readyTxt := "waiting for player"
	buttonTxt := "I'm Ready!"
	inputTxt := "ready"
	isReady := r.FormValue("ready") == "ready"
	player := room.Game.Players.SetReady(idCookie.Value, isReady)

	if isReady {
		readyTxt = "Let's Go!"
		buttonTxt = "Wait! I'm not ready!"
		inputTxt = "wait"
		if room.Game.Players.AllReady() {
			room.Game.SetScene(gamedata.SceneCombat)

			data := room.Game.Get(idCookie.Value)
			var buf bytes.Buffer
			if err := s.Tmpl.ExecuteTemplate(&buf, "gameContent", data); err != nil {
				log.Println(err)
				http.Error(w, "Error executing game template", http.StatusInternalServerError)
			}

			countdownStart := room.Game.Config.CountdownSecs
			var countdownBuf bytes.Buffer
			if err := s.Tmpl.ExecuteTemplate(&countdownBuf, "lobbyCountdown", countdownStart); err != nil {
				log.Println(err)
				http.Error(w, "Error executing lobbyCountdown template", http.StatusInternalServerError)
			}
			countdownOOB := fmt.Sprintf(`<div id="lobby" hx-swap-oob="afterend">%s</div>`, countdownBuf.String())
			room.Broadcaster.BroadcastOOB("swap", countdownOOB)

			go func() {
				for i := range countdownStart {
					room.Broadcaster.BroadcastOOB("swap", fmt.Sprintf(`<span id="countdown_num" hx-swap-oob="true">%d</span>`, countdownStart-i))
					time.Sleep(1 * time.Second)
				}

				room.Game.StartRound()

				// Create game record in DB
				if s.DB != nil {
					gameID, err := s.DB.CreateGame(room.Code, room.HostID, room.Game.Config.RoundDuration*1000)
					if err != nil {
						log.Printf("[DB] CreateGame error: %v\n", err)
					} else {
						room.Game.SetCurrentGameID(gameID)
					}
				}

				data := room.Game.Get(idCookie.Value)
				var gameBuf bytes.Buffer
				if err := s.Tmpl.ExecuteTemplate(&gameBuf, "gameContent", data); err != nil {
					log.Println(err)
					return
				}
				gameOOB := fmt.Sprintf(`<div id="scene" hx-swap-oob="innerHTML">%s</div>`, gameBuf.String())
				room.Broadcaster.BroadcastOOB("swap", gameOOB)

				s.startRoundTimer(room)
			}()

			return
		}
	}

	playerOOB := fmt.Sprintf(`<div id="lobby_player_ready%s" hx-swap-oob="innerHTML">%s</div>`, player.ID, readyTxt)
	room.Broadcaster.BroadcastOOB("swap", playerOOB)

	buttonOOB := fmt.Sprintf(`<button id="ready_button" hx-swap-oob="innerHTML">%s</button>`, buttonTxt)
	inputOOB := fmt.Sprintf(`<input id="ready_input" type="hidden" name="ready" hx-swap-oob="outerHTML" value="%s"/>`, inputTxt)
	if _, err := w.Write([]byte(buttonOOB + inputOOB)); err != nil {
		log.Println(err)
	}
}

func (s *Server) startRoundTimer(room *rooms.Room) {
	duration := room.Game.Config.RoundDuration
	for i := duration; i >= 0; i-- {
		room.Game.SetTimeLeft(i)
		room.Broadcaster.BroadcastOOB("swap", fmt.Sprintf(`<div id="timer" hx-swap-oob="innerHTML">%d</div>`, i))
		if i == 0 {
			break
		}
		time.Sleep(1 * time.Second)
	}

	rankings := room.Game.EndRound()

	// Persist game results and award badges
	if s.DB != nil {
		gameID := room.Game.CurrentGameID()
		if gameID != "" {
			if err := s.DB.EndGame(gameID); err != nil {
				log.Printf("[DB] EndGame error: %v\n", err)
			}
			for i, p := range rankings {
				if err := s.DB.AddGamePlayer(gameID, p.ID, p.Score, i+1); err != nil {
					log.Printf("[DB] AddGamePlayer error: %v\n", err)
				}
			}
			// Award badges
			q := analytics.NewQueries(s.DB)
			for _, p := range rankings {
				gameStats, err := q.GetPlayerGameStats(gameID, p.ID)
				if err != nil {
					log.Printf("[DB] GetPlayerGameStats error: %v\n", err)
					continue
				}
				gameBadges := analytics.EvaluateGameBadges(*gameStats)
				for _, b := range gameBadges {
					gID := gameID
					if err := s.DB.AwardBadge(p.ID, string(b.ID), &gID); err != nil {
						log.Printf("[DB] AwardBadge error: %v\n", err)
					}
				}
				// Check lifetime badges
				lifeStats, err := q.GetPlayerLifetimeStats(p.ID)
				if err == nil {
					lifeBadges := analytics.EvaluateLifetimeBadges(*lifeStats)
					for _, b := range lifeBadges {
						if err := s.DB.AwardBadge(p.ID, string(b.ID), nil); err != nil {
							log.Printf("[DB] AwardBadge error: %v\n", err)
						}
					}
				}
			}
		}
	}

	var buf bytes.Buffer
	if err := s.Tmpl.ExecuteTemplate(&buf, "recap", rankings); err != nil {
		log.Println(err)
		return
	}
	recapOOB := fmt.Sprintf(`<div id="scene" hx-swap-oob="innerHTML">%s</div>`, buf.String())
	room.Broadcaster.BroadcastOOB("swap", recapOOB)
}

func (s *Server) handlePoll(w http.ResponseWriter, r *http.Request) {
	room := s.getRoom(r)
	if room == nil {
		http.Error(w, "Room not found", http.StatusBadRequest)
		return
	}

	idCookie, err := r.Cookie("player_id")
	if err != nil {
		http.Error(w, "Not Registered", http.StatusBadRequest)
		return
	}
	if err := s.Tmpl.ExecuteTemplate(w, "gameContent", room.Game.Get(idCookie.Value)); err != nil {
		log.Println(err)
	}
}

func (s *Server) handleTarget(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[Handle:Target] Request Received")
	room := s.getRoom(r)
	if room == nil {
		http.Error(w, "Room not found", http.StatusBadRequest)
		return
	}

	idCookie, err := r.Cookie("player_id")
	if err != nil {
		http.Error(w, "Not Registered", http.StatusBadRequest)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	// /room/target/{id}/{points} -> parts: ["", "room", "target", id, points]
	if len(parts) < 5 {
		http.Error(w, "Invalid target path", http.StatusBadRequest)
		return
	}

	strTargetId := parts[3]
	targetId, err := strconv.Atoi(strTargetId)
	if err != nil {
		log.Println(err)
		http.Error(w, "Invalid target ID", http.StatusBadRequest)
		return
	}
	// Capture target info before killing for click recording
	target := room.Game.Targets.Get(targetId)
	clickedAt := time.Now()

	if !room.Game.Targets.Kill(targetId) {
		// Target already dead — ignore duplicate click
		w.WriteHeader(http.StatusOK)
		return
	}
	time.AfterFunc(500*time.Millisecond, func() {
		newTarget := room.Game.Targets.Add()
		var buf bytes.Buffer
		if err := s.Tmpl.ExecuteTemplate(&buf, "target", newTarget); err != nil {
			log.Println(err)
		}
		room.Broadcaster.BroadcastOOB("newTarget", buf.String())
	})

	strPoints := parts[4]
	points, err := strconv.Atoi(strPoints)
	if err != nil {
		log.Println(err)
		http.Error(w, "Invalid points", http.StatusBadRequest)
		return
	}
	playerId := idCookie.Value
	player := room.Game.Players.UpdateScore(playerId, points)

	// Record click event asynchronously
	if s.ClickBuffer != nil && target != nil {
		gameID := room.Game.CurrentGameID()
		if gameID != "" {
			reactionMs := int(clickedAt.Sub(target.SpawnedAt).Milliseconds())
			select {
			case s.ClickBuffer <- db.ClickEvent{
				GameID:     gameID,
				PlayerID:   playerId,
				TargetID:   targetId,
				Points:     points,
				TargetSize: target.Size,
				TargetX:    target.X,
				TargetY:    target.Y,
				SpawnedAt:  target.SpawnedAt,
				ClickedAt:  clickedAt,
				ReactionMs: reactionMs,
			}:
			default:
				log.Println("[DB] Click buffer full, dropping event")
			}
		}
	}

	targetOOB := fmt.Sprintf(`<div id="target_%d" hx-swap-oob="delete"></div>`, targetId)
	playerOOB := fmt.Sprintf(`<div id="player_score_%s" hx-swap-oob="innerHTML">%d</div>`, player.ID, player.Score)
	room.Broadcaster.BroadcastOOB("swap", targetOOB+playerOOB)
}

func (s *Server) handlePlayAgain(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[Handle:PlayAgain] Request Received")
	room := s.getRoom(r)
	if room == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	room.Game.ResetToLobby()

	idCookie, err := r.Cookie("player_id")
	if err != nil {
		http.Redirect(w, r, "/room/"+room.Code, http.StatusSeeOther)
		return
	}

	data := room.Game.Get(idCookie.Value)
	data.RoomCode = room.Code
	var buf bytes.Buffer
	if err := s.Tmpl.ExecuteTemplate(&buf, "lobby", data); err != nil {
		log.Println(err)
	}
	lobbyOOB := fmt.Sprintf(`<div id="scene" hx-swap-oob="innerHTML">%s</div>`, buf.String())
	room.Broadcaster.BroadcastOOB("swap", lobbyOOB)
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	room := s.getRoom(r)
	if room == nil {
		http.Error(w, "Room not found", http.StatusBadRequest)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	msgChan := room.Broadcaster.Subscribe()
	defer room.Broadcaster.Unsubscribe(msgChan)

	for {
		select {
		case <-r.Context().Done():
			return
		case msg := <-msgChan:
			fmt.Fprintf(w, "event: %s\n", msg.Event)
			for _, line := range strings.Split(msg.Msg, "\n") {
				fmt.Fprintf(w, "data: %s\n", line)
			}
			fmt.Fprint(w, "\n")
			flusher.Flush()
		}
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	status := "ok"
	if s.DB != nil {
		if err := s.DB.Ping(); err != nil {
			status = "db_error"
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"status":"%s","error":"%s"}`, status, err.Error())
			return
		}
	}
	fmt.Fprintf(w, `{"status":"%s"}`, status)
}
