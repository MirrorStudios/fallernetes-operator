package kubeport

import (
	"context"

	"github.com/MirrorStudios/fallernetes-operator/service/internal/gen"
)

type FleetPort interface {
	CreateFleet(ctx context.Context, name, namespace string, labels map[string]string, spec gen.FleetSpec) error
	DeleteFleet(ctx context.Context, name, namespace string, force bool) error
	PatchFleetReplicas(ctx context.Context, name, namespace string, replicas int32, minReplicas, maxReplicas *int32) error
}
