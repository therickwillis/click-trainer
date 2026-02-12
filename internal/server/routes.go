package server

import (
	"fmt"
	"net/http"
	"text/template"
)

var (
	tmpl *template.Template
)

func Run() error {
	tmpl = template.Must(template.ParseFiles("templates/game.html", "templates/join.html", "templates/target.html", "templates/lobby.html"))

	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/register", handleRegister)
	http.HandleFunc("/ready", handleReady)
	http.HandleFunc("/target/", handleTarget)
	http.HandleFunc("/events", handleEvents)
	http.HandleFunc("/poll", handlePoll)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	fmt.Println("Server listening on http://localhost:8080")
	return http.ListenAndServe("0.0.0.0:8080", nil)
}
