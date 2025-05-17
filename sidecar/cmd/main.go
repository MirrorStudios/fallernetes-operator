package main

import (
	"github.com/MirrorStudios/fallernetes-sidecar/internal/app"
	"github.com/MirrorStudios/fallernetes-sidecar/internal/routes"
	"net/http"
)

func main() {
	a := app.App{
		Mux:               http.NewServeMux(),
		ShutdownRequested: false,
		DeleteAllowed:     false,
	}

	routes.SetupRoutes(&a)
}
