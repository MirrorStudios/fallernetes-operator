package utils

import (
	"context"
	"github.com/MirrorStudios/fallernetes/api/v1alpha1"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetFleetsForType(ctx context.Context, c client.Client, gametype *v1alpha1.GameType, logger logr.Logger) (*v1alpha1.FleetList, error) {
	fleetList := &v1alpha1.FleetList{}

	labelSelector := client.MatchingLabels{
		"gametype": gametype.Name,
	}

	if err := c.List(ctx, fleetList, labelSelector); err != nil {
		logger.Error(err, "Failed to list Fleets", "GameType", gametype.Name)
		return nil, err
	}

	return fleetList, nil
}

func GetFleetObjectForType(gametype *v1alpha1.GameType) *v1alpha1.Fleet {
	labels := gametype.Labels
	if labels == nil {
		labels = map[string]string{}
	}
	labels["gametype"] = gametype.Name

	fleet := &v1alpha1.Fleet{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: gametype.Name + "-",
			Namespace:    gametype.Namespace,
			Labels:       labels,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(gametype, v1alpha1.GroupVersion.WithKind("GameType")),
			},
		},
		Spec: gametype.Spec.FleetSpec,
	}

	return fleet
}
