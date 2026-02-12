package server

import (
	"bytes"
	gamedata "clicktrainer/internal/gamedata"
	"clicktrainer/internal/players"
	"clicktrainer/internal/targets"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

func handleIndex(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[Handle:Index] Request Received")

	idCookie, err := r.Cookie("player_id")
	if err == nil {
		nameCookie, err := r.Cookie("player_name")
		if err != nil {
			http.SetCookie(w, &http.Cookie{
				Name:   "player_id",
				MaxAge: -1,
			})

			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		if !players.ValidateSession(idCookie.Value) {
			player := players.Add(idCookie.Value, nameCookie.Value)
			var buf bytes.Buffer
			if err := tmpl.ExecuteTemplate(&buf, "lobbyPlayer", player); err != nil {
				log.Println(err)
			}
			fmt.Println("[Handle:Index] New Player OOB Broadcast")
			BroadcastOOB("newPlayer", buf.String())
		}

		gameData := gamedata.Get(idCookie.Value)

		if err := tmpl.ExecuteTemplate(w, "game", gameData); err != nil {
			fmt.Println(err.Error())
			http.Error(w, "Error rendering game view", http.StatusInternalServerError)
		}
		return
	}

	if err := tmpl.ExecuteTemplate(w, "join", nil); err != nil {
		fmt.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleReady(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[Handle:Ready] Request Received")
	idCookie, err := r.Cookie("player_id")
	if err != nil {
		http.Error(w, "Not Registered", http.StatusBadRequest)
		return
	}

	readyTxt := "waiting for player"
	buttonTxt := "I'm Ready!"
	inputTxt := "ready"
	isReady := r.FormValue("ready") == "ready"
	player := players.SetReady(idCookie.Value, isReady)

	if isReady {
		readyTxt = "Let's Go!"
		buttonTxt = "Wait! I'm not ready!"
		inputTxt = "wait"
		if players.AllReady() {
			gamedata.SetScene(gamedata.SceneCombat)

			data := gamedata.Get(idCookie.Value)
			var buf bytes.Buffer
			if err := tmpl.ExecuteTemplate(&buf, "gameContent", data); err != nil {
				fmt.Println(err.Error())
				http.Error(w, "Error executing game template", http.StatusInternalServerError)
			}

			countdownStart := 2
			var countdownBuf bytes.Buffer
			if err := tmpl.ExecuteTemplate(&countdownBuf, "lobbyCountdown", countdownStart); err != nil {
				fmt.Println(err.Error())
				http.Error(w, "Error executing lobbyCountdown template", http.StatusInternalServerError)
			}
			countdownOOB := fmt.Sprintf(`<div id="lobby" hx-swap-oob="afterend">%s</div>`, countdownBuf.String())
			BroadcastOOB("swap", countdownOOB)

			go func() {
				for i := range countdownStart {
					BroadcastOOB("swap", fmt.Sprintf(`<span id="countdown_num" hx-swap-oob="true">%d</span>`, countdownStart-i))
					time.Sleep(1 * time.Second)
				}
				gameOOB := fmt.Sprintf(`<div id="scene" hx-swap-oob="innerHTML">%s</div>`, buf.String())
				BroadcastOOB("swap", gameOOB)
			}()

			return
		}
	}

	playerOOB := fmt.Sprintf(`<div id="lobby_player_ready%s" hx-swap-oob="innerHTML">%s</div>`, player.ID, readyTxt)
	BroadcastOOB("swap", playerOOB)

	buttonOOB := fmt.Sprintf(`<button id="ready_button" hx-swap-oob="innerHTML">%s</button>`, buttonTxt)
	inputOOB := fmt.Sprintf(`<input id="ready_input" type="hidden" name="ready" hx-swap-oob="outerHTML" value="%s"/>`, inputTxt)
	if _, err := w.Write([]byte(buttonOOB + inputOOB)); err != nil {
		fmt.Println(err.Error())
	}
}

func handlePoll(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[Handle:Poll] Request Received")
	idCookie, err := r.Cookie("player_id")
	if err != nil {
		http.Error(w, "Not Registered", http.StatusBadRequest)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "gameContent", gamedata.Get(idCookie.Value)); err != nil {
		log.Println(err)
	}
}

func handleTarget(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[Handle:Target] Request Received")
	idCookie, err := r.Cookie("player_id")
	if err != nil {
		http.Error(w, "Not Registered", http.StatusBadRequest)
		return
	}
	parts := strings.Split(r.URL.Path, "/")

	// Target
	strTargetId := parts[2]
	targetId, err := strconv.Atoi(strTargetId)
	if err != nil {
		log.Println(err)
	}
	targets.Kill(targetId)
	time.AfterFunc(500*time.Millisecond, func() {
		newTarget := targets.Add()
		var buf bytes.Buffer
		if err := tmpl.ExecuteTemplate(&buf, "target", newTarget); err != nil {
			log.Println(err)
		}

		fmt.Println("sending OOB for new Target")
		BroadcastOOB("newTarget", buf.String())
	})

	// Player
	strPoints := parts[3]
	points, err := strconv.Atoi(strPoints)
	if err != nil {
		log.Println(err)
	}
	playerId := idCookie.Value
	player := players.UpdateScore(playerId, points)

	targetOOB := fmt.Sprintf(`<div id="target_%d" hx-swap-oob="delete"></div>`, targetId)
	playerOOB := fmt.Sprintf(`<div id="player_score_%s" hx-swap-oob="innerHTML">%d</div>`, player.ID, player.Score)
	BroadcastOOB("swap", targetOOB+playerOOB)
}

func handleRegister(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[Handle:Register] Request Received")
	if err := r.ParseForm(); err != nil {
		fmt.Println(err.Error())
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

	players.Add(id, name)

	data := gamedata.Get(id)

	switch data.Scene {
	case gamedata.SceneLobby:
		var buf bytes.Buffer
		if err := tmpl.ExecuteTemplate(&buf, "lobbyPlayer", data.Player); err != nil {
			log.Println(err)
		}

		BroadcastOOB("newPlayer", buf.String())
	default:
		var buf bytes.Buffer
		if err := tmpl.ExecuteTemplate(&buf, "scoreboard", players.GetList()); err != nil {
			log.Println(err)
		}
		BroadcastOOB("scoreboard", buf.String())
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
