package service

import (
	"context"
	"log/slog"

	"github.com/MirrorStudios/fallernetes-sidecar/internal/gen"
)

func (s *SidecarService) GetShutdown(ctx context.Context, req gen.GetShutdownRequestObject) (gen.GetShutdownResponseObject, error) {
	return gen.GetShutdown200JSONResponse{Shutdown: s.state.IsShutdownRequested()}, nil
}

func (s *SidecarService) SetShutdown(ctx context.Context, req gen.SetShutdownRequestObject) (gen.SetShutdownResponseObject, error) {
	shutdown := req.Body.Shutdown
	if s.state.IsShutdownRequested() != shutdown {
		slog.Info("Shutdown will be updated", "current", s.state.IsShutdownRequested(), "new", shutdown)
	}
	s.state.SetShutdownRequested(shutdown)
	return gen.SetShutdown200Response{}, nil
}
