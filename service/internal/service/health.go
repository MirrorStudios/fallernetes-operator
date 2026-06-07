package service

import (
	"context"

	"github.com/MirrorStudios/fallernetes-service/internal/gen"
)

func (s *OperatorService) Health(ctx context.Context, req gen.HealthRequestObject) (gen.HealthResponseObject, error) {
	return gen.Health200Response{}, nil
}
