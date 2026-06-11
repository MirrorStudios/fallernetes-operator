package app

import (
	"log/slog"
	"net/http"
	"sync/atomic"

	"github.com/MirrorStudios/fallernetes-operator/sidecar/internal/service"
)

// App struct is where most of the state of the sidecar is stored, along with the used http Mux.
type App struct {
	Mux               *http.ServeMux
	DeleteAllowed     atomic.Bool
	ShutdownRequested atomic.Bool
	Port              int
	Logger            *slog.Logger
	Service           *service.SidecarService
}
