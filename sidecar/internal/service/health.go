package service

import (
	"context"

	"github.com/MirrorStudios/fallernetes-operator/sidecar/internal/gen"
)

func (s *SidecarService) Health(ctx context.Context, req gen.HealthRequestObject) (gen.HealthResponseObject, error) {
	return gen.Health200Response{}, nil
}
