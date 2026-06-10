package state

type StatePort interface {
	IsDeleteAllowed() bool
	SetDeleteAllowed(allowed bool)
	IsShutdownRequested() bool
	SetShutdownRequested(shutdown bool)
}
