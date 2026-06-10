package service

import (
	"context"
	"log/slog"

	"github.com/MirrorStudios/fallernetes-sidecar/internal/gen"
)

func (s *SidecarService) GetAllowDelete(ctx context.Context, req gen.GetAllowDeleteRequestObject) (gen.GetAllowDeleteResponseObject, error) {
	return gen.GetAllowDelete200JSONResponse{Allowed: s.state.IsDeleteAllowed()}, nil
}

func (s *SidecarService) SetAllowDelete(ctx context.Context, req gen.SetAllowDeleteRequestObject) (gen.SetAllowDeleteResponseObject, error) {
	allowed := req.Body.Allowed
	if s.state.IsDeleteAllowed() != allowed {
		slog.Info("Allowed will be updated", "current", s.state.IsDeleteAllowed(), "new", allowed)
	}
	s.state.SetDeleteAllowed(allowed)
	return gen.SetAllowDelete200Response{}, nil
}
