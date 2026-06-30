package main

import (
	"github.com/knowledgeos/backend/internal/app"
	"github.com/knowledgeos/backend/internal/config"
	"github.com/rs/zerolog/log"
)

func main() {
	cfg := config.Load()
	application, err := app.New(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize application")
	}
	if err := application.Run(); err != nil {
		log.Fatal().Err(err).Msg("application stopped with error")
	}
}
