package main

import (
	"fmt"
	"github.com/MirrorStudios/fallernetes-sidecar/internal/app"
	"github.com/MirrorStudios/fallernetes-sidecar/internal/routes"
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

	a := app.App{
		Mux:               http.NewServeMux(),
		ShutdownRequested: false,
		DeleteAllowed:     false,
		Port:              port,
	}

	routes.SetupRoutes(&a)
}
