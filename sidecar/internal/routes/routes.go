package routes

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/MirrorStudios/fallernetes-operator/sidecar/internal/app"
	"github.com/MirrorStudios/fallernetes-operator/sidecar/internal/gen"
)

func SetupRoutes(a *app.App) {
	strictHandler := gen.NewStrictHandler(a.Service, nil)
	gen.HandlerFromMux(strictHandler, a.Mux)

	srv := &http.Server{
		Addr:    ":" + strconv.Itoa(a.Port),
		Handler: app.LogRoute(a, a.Mux),
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	a.Logger.Info("Starting http server", "port", a.Port)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting server: %v", err)
		}
	}()

	<-ctx.Done()
	a.Logger.Info("Shutting down server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}
	a.Logger.Info("Server stopped")
}
