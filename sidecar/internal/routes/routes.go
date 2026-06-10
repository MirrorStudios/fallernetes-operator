package routes

import (
	"log"
	"net/http"
	"strconv"

	"github.com/MirrorStudios/fallernetes-sidecar/internal/app"
	"github.com/MirrorStudios/fallernetes-sidecar/internal/gen"
)

func SetupRoutes(a *app.App) {
	strictHandler := gen.NewStrictHandler(a.Service, nil)
	gen.HandlerFromMux(strictHandler, a.Mux)

	loggingHandler := app.LogRoute(a, a.Mux)
	a.Logger.Info("Starting http server", "port", a.Port)
	if err := http.ListenAndServe(":"+strconv.Itoa(a.Port), loggingHandler); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
