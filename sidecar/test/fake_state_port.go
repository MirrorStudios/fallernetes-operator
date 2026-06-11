package test

import stateport "github.com/MirrorStudios/fallernetes-operator/sidecar/internal/ports/state"

type FakeStatePort struct {
	deleteAllowed     bool
	shutdownRequested bool
}

var _ stateport.StatePort = &FakeStatePort{}

func (f *FakeStatePort) IsDeleteAllowed() bool          { return f.deleteAllowed }
func (f *FakeStatePort) SetDeleteAllowed(allowed bool)  { f.deleteAllowed = allowed }
func (f *FakeStatePort) IsShutdownRequested() bool      { return f.shutdownRequested }
func (f *FakeStatePort) SetShutdownRequested(s bool)    { f.shutdownRequested = s }
