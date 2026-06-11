package state

import "github.com/MirrorStudios/fallernetes-operator/sidecar/internal/app"

type StateAdapter struct {
	app *app.App
}

func NewStateAdapter(a *app.App) *StateAdapter {
	return &StateAdapter{app: a}
}

func (s *StateAdapter) IsDeleteAllowed() bool {
	return s.app.DeleteAllowed.Load()
}

func (s *StateAdapter) SetDeleteAllowed(allowed bool) {
	s.app.DeleteAllowed.Store(allowed)
}

func (s *StateAdapter) IsShutdownRequested() bool {
	return s.app.ShutdownRequested.Load()
}

func (s *StateAdapter) SetShutdownRequested(shutdown bool) {
	s.app.ShutdownRequested.Store(shutdown)
}
