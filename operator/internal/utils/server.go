package utils

import (
	"context"

	"github.com/MirrorStudios/fallernetes-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetServersForFleet lists all Server objects owned by the given fleet.
func GetServersForFleet(ctx context.Context, c client.Client, fleet *v1alpha1.Fleet) (*v1alpha1.ServerList, error) {
	serverList := &v1alpha1.ServerList{}
	if err := c.List(ctx, serverList, client.InNamespace(fleet.Namespace), client.MatchingLabels{"fleet": fleet.Name}); err != nil {
		return nil, err
	}
	return serverList, nil
}
