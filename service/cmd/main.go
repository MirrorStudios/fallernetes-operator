package main

import (
	"github.com/MirrorStudios/fallernetes-service/internal/app"
	"github.com/MirrorStudios/fallernetes-service/internal/routes"
)

func main() {
	a := app.CreateApp()
	routes.SetupRoutes(a)
}
