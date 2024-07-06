package main

import (
	"github.com/zhukovrost/pasteAPI-email-sender/internal/app"
	"github.com/zhukovrost/pasteAPI-email-sender/internal/config"
	"log"
)

func main() {
	cfg, err := config.New()
	if err != nil {
		log.Fatalf("Config error: %s", err)
	}

	app.Run(cfg)
}
