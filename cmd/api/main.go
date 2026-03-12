package main

import (
	"log"

	"cep-api/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatalf("failed to start application: %v", err)
	}
}
