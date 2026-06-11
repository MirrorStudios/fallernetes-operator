package main

import (
	"github.com/MirrorStudios/fallernetes-operator/service/internal/app"
	"github.com/MirrorStudios/fallernetes-operator/service/internal/routes"
)

func main() {
	a := app.CreateApp()
	routes.SetupRoutes(a)
}
