package main

import (
	"log"

	"github.com/mirjalilova/voice_transcribe/config"
	"github.com/mirjalilova/voice_transcribe/internal/app"
)

func main() {

	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Config error: %s", err)
	}

	// Run
	app.Run(cfg)
}
