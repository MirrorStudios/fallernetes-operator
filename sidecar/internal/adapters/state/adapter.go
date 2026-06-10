package state

import "github.com/MirrorStudios/fallernetes-sidecar/internal/app"

type StateAdapter struct {
	app *app.App
}

func NewStateAdapter(a *app.App) *StateAdapter {
	return &StateAdapter{app: a}
}

func (s *StateAdapter) IsDeleteAllowed() bool {
	return s.app.DeleteAllowed
}

func (s *StateAdapter) SetDeleteAllowed(allowed bool) {
	s.app.DeleteAllowed = allowed
}

func (s *StateAdapter) IsShutdownRequested() bool {
	return s.app.ShutdownRequested
}

func (s *StateAdapter) SetShutdownRequested(shutdown bool) {
	s.app.ShutdownRequested = shutdown
}
