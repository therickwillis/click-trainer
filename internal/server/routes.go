package server

import (
	"clicktrainer/internal/config"
	"clicktrainer/internal/db"
	"clicktrainer/internal/gamedata"
	"clicktrainer/internal/rooms"
	"fmt"
	"log"
	"net/http"
	"text/template"
	"time"
)

func Run() error {
	appCfg := config.Load()

	gameCfg := gamedata.Config{
		RoundDuration:  appCfg.RoundDuration,
		InitialTargets: 3,
		CountdownSecs:  3,
	}
	roomStore := rooms.NewStore(gameCfg)

	funcMap := template.FuncMap{
		"inc": func(i int) int { return i + 1 },
	}
	tmpl := template.Must(template.New("").Funcs(funcMap).ParseFiles(
		"templates/home.html",
		"templates/game.html",
		"templates/join.html",
		"templates/target.html",
		"templates/lobby.html",
		"templates/recap.html",
		"templates/analytics/dashboard.html",
		"templates/analytics/leaderboard.html",
		"templates/analytics/player.html",
		"templates/analytics/game.html",
	))

	srv := &Server{
		Rooms: roomStore,
		Tmpl:  tmpl,
	}

	// Optional database connection
	if appCfg.DatabaseURL != "" {
		database, err := db.Connect(appCfg.DatabaseURL)
		if err != nil {
			log.Printf("[DB] Failed to connect: %v (running without database)\n", err)
		} else {
			if err := database.Migrate(); err != nil {
				log.Printf("[DB] Migration failed: %v\n", err)
			}
			srv.DB = database
			srv.ClickBuffer = make(chan db.ClickEvent, 1000)
			go clickBatchWriter(database, srv.ClickBuffer)
			log.Println("[DB] Database connected and migrations applied")
		}
	} else {
		log.Println("[DB] DATABASE_URL not set, running without database")
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
	mux.HandleFunc("/health", srv.handleHealth)
	mux.HandleFunc("/analytics", srv.handleAnalyticsDashboard)
	mux.HandleFunc("/analytics/leaderboard", srv.handleAnalyticsLeaderboard)
	mux.HandleFunc("/analytics/player/", srv.handleAnalyticsPlayer)
	mux.HandleFunc("/analytics/game/", srv.handleAnalyticsGame)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	addr := "0.0.0.0:" + appCfg.Port
	fmt.Printf("Server listening on http://localhost:%s\n", appCfg.Port)
	return http.ListenAndServe(addr, mux)
}

func clickBatchWriter(database *db.DB, buffer chan db.ClickEvent) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	batch := make([]db.ClickEvent, 0, 50)

	for {
		select {
		case ev := <-buffer:
			batch = append(batch, ev)
			if len(batch) >= 50 {
				if err := database.BatchRecordClicks(batch); err != nil {
					log.Printf("[DB] BatchRecordClicks error: %v\n", err)
				}
				batch = batch[:0]
			}
		case <-ticker.C:
			if len(batch) > 0 {
				if err := database.BatchRecordClicks(batch); err != nil {
					log.Printf("[DB] BatchRecordClicks error: %v\n", err)
				}
				batch = batch[:0]
			}
		}
	}
}
