package main

import (
	"log"

	"codex-switch/internal/classicapp"
	"codex-switch/internal/profiles"
)

func main() {
	store, err := profiles.NewDefaultStore()
	if err != nil {
		log.Fatalf("initialize profile store: %v", err)
	}

	trayApp := classicapp.NewTrayApp(store)
	trayApp.Run()
}
