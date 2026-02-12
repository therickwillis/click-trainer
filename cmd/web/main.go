package main

import (
	"clicktrainer/internal/server"
	"log"
)

func main() {
	if err := server.Run(); err != nil {
		log.Fatal(err.Error())
	}
}
