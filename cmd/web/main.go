package main

import (
	"clicktrainer/internal/server"
	"log"
	"log/slog"
	"os"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := server.Run(); err != nil {
		log.Fatal(err.Error())
	}
}
