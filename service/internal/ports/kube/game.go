package kubeport

import (
	"context"

	"github.com/MirrorStudios/fallernetes-service/internal/gen"
)

type GamePort interface {
	CreateGame(ctx context.Context, name, namespace string, labels map[string]string, spec gen.GameTypeSpec) error
	DeleteGame(ctx context.Context, name, namespace string, force bool) error
}
