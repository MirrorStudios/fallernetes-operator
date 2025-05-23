package routes

import (
	"github.com/MirrorStudios/fallernetes-sidecar/internal/app"
	"github.com/MirrorStudios/fallernetes-sidecar/internal/handlers"
	"log"
	"net/http"
	"strconv"
)

// SetupRoutes sets up the nessecary routes, their handlers and starts serving http.
func SetupRoutes(a *app.App) {

	a.Mux.HandleFunc("GET /allow_delete", handlers.IsDeleteAllowed(a))
	a.Mux.HandleFunc("POST /allow_delete", handlers.SetDeleteAllowed(a))
	a.Mux.HandleFunc("GET /shutdown", handlers.IsShutdownRequested(a))
	a.Mux.HandleFunc("POST /shutdown", handlers.SetShutdownRequested(a))
	a.Mux.HandleFunc("/health", handlers.Health(a))
	loggingHandler := app.LogRoute(a, a.Mux)
	a.Logger.Info("Starting http server", "port", a.Port)
	err := http.ListenAndServe(":"+strconv.Itoa(a.Port), loggingHandler)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
