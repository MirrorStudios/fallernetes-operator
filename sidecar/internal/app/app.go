package app

import (
	"log/slog"
	"net/http"

	"github.com/MirrorStudios/fallernetes-sidecar/internal/service"
)

// App struct is where most of the state of the sidecar is stored, along with the used http Mux.
type App struct {
	Mux               *http.ServeMux
	DeleteAllowed     bool
	ShutdownRequested bool
	Port              int
	Logger            *slog.Logger
	Service           *service.SidecarService
}
