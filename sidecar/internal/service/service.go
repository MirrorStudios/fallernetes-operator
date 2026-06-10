package service

import stateport "github.com/MirrorStudios/fallernetes-sidecar/internal/ports/state"

type SidecarService struct {
	state stateport.StatePort
}

func NewSidecarService(state stateport.StatePort) *SidecarService {
	return &SidecarService{state: state}
}
