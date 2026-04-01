package main

import (
	"log"

	"codex-switch/internal/app"
	"codex-switch/internal/profiles"
)

func main() {
	store, err := profiles.NewDefaultStore()
	if err != nil {
		log.Fatalf("initialize profile store: %v", err)
	}

	trayApp := app.NewTrayApp(store)
	trayApp.Run()
}
