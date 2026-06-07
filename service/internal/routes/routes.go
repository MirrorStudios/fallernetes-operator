package routes

import (
	"log"
	"net/http"

	"github.com/MirrorStudios/fallernetes-service/internal/app"
	"github.com/MirrorStudios/fallernetes-service/internal/gen"
)

func SetupRoutes(a *app.App) {
	strictHandler := gen.NewStrictHandler(a.Service, nil)
	gen.HandlerFromMux(strictHandler, a.Mux)

	if err := http.ListenAndServe(":8080", a.Mux); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
