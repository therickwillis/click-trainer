package main

import (
	"clicktrainer/internal/server"
	"clicktrainer/internal/targets"
	"log"
)

func main() {
	for i := 0; i < 3; i++ {
		targets.Add()
	}

	if err := server.Run(); err != nil {
		log.Fatal(err.Error())
	}
}
