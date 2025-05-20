package main

import (
	"fmt"
	"github.com/MirrorStudios/fallernetes-sidecar/internal/app"
	"github.com/MirrorStudios/fallernetes-sidecar/internal/routes"
	"log/slog"
	"net/http"
	"os"
	"strconv"
)

func main() {
	var port int
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

	a := app.App{
		Mux:               http.NewServeMux(),
		ShutdownRequested: false,
		DeleteAllowed:     false,
		Port:              port,
		Logger:            logger,
	}

	routes.SetupRoutes(&a)
}

func isDebug() bool {
	if os.Getenv("DEBUG") == "true" {
		return true
	}
	return false
}
