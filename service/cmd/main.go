package main

import (
	"log/slog"
	"os"

	"github.com/MirrorStudios/fallernetes-operator/service/internal/app"
	"github.com/MirrorStudios/fallernetes-operator/service/internal/routes"
)

func main() {
	level := slog.LevelInfo
	if os.Getenv("DEBUG") == "true" {
		level = slog.LevelDebug
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	logger = logger.With("api", "service")
	slog.SetDefault(logger)

	a := app.CreateApp(logger)
	routes.SetupRoutes(a)
}
