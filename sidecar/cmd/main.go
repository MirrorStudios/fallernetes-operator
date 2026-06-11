package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/MirrorStudios/fallernetes-operator/sidecar/internal/adapters/state"
	"github.com/MirrorStudios/fallernetes-operator/sidecar/internal/app"
	"github.com/MirrorStudios/fallernetes-operator/sidecar/internal/routes"
	"github.com/MirrorStudios/fallernetes-operator/sidecar/internal/service"
)

func main() {
	portStr := os.Getenv("PORT")
	if portStr == "" {
		fmt.Println("PORT environment variable not set, defaulting to 8080.")
		portStr = "8080"
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		fmt.Printf("Invalid port value: %v\n", err)
		return
	}

	level := slog.LevelInfo
	if isDebug() {
		level = slog.LevelDebug
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	logger = logger.With("api", "sidecar")
	slog.SetDefault(logger)

	a := &app.App{
		Mux:    http.NewServeMux(),
		Port:   port,
		Logger: logger,
	}

	adapter := state.NewStateAdapter(a)
	a.Service = service.NewSidecarService(adapter)

	routes.SetupRoutes(a)
}

func isDebug() bool {
	return os.Getenv("DEBUG") == "true"
}
