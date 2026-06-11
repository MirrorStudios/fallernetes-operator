package kubeport

import (
	"context"

	"github.com/MirrorStudios/fallernetes-operator/service/internal/gen"
)

type ScalerPort interface {
	CreateScaler(ctx context.Context, name, namespace string, labels map[string]string, spec gen.GameAutoscalerSpec) error
	DeleteScaler(ctx context.Context, name, namespace string) error
}
