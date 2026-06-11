package kubeport

import (
	"context"

	"github.com/MirrorStudios/fallernetes-operator/service/internal/gen"
)

type ServerPort interface {
	CreateServer(ctx context.Context, name, namespace string, labels map[string]string, spec gen.ServerSpec) error
	DeleteServer(ctx context.Context, name, namespace string, force bool) error
}
