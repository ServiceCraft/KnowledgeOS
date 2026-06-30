package main

import (
	"github.com/knowledgeos/backend/internal/config"
	"github.com/knowledgeos/backend/internal/database"
	applog "github.com/knowledgeos/backend/internal/logger"
	"github.com/rs/zerolog/log"
)

func main() {
	cfg := config.Load()
	applog.Configure(cfg.LogLevel, cfg.LogFormat, "knowledgeos-migrate")

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	if err := database.RunMigrations(db, "migrations"); err != nil {
		log.Fatal().Err(err).Msg("failed to run migrations")
	}
	log.Info().Msg("migrations applied")
}
