package utils

import (
	"context"

	"github.com/MirrorStudios/fallernetes-operator/api/v1alpha1"
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

// GetOldestFleet returns the fleet with the earliest CreationTimestamp, or nil for an empty slice.
func GetOldestFleet(fleets []v1alpha1.Fleet) *v1alpha1.Fleet {
	var oldest *v1alpha1.Fleet
	for i := range fleets {
		if oldest == nil || fleets[i].CreationTimestamp.Before(&oldest.CreationTimestamp) {
			oldest = &fleets[i]
		}
	}
	return oldest
}

// GetNewestFleet returns the fleet with the latest CreationTimestamp, or nil for an empty slice.
func GetNewestFleet(fleets []v1alpha1.Fleet) *v1alpha1.Fleet {
	var newest *v1alpha1.Fleet
	for i := range fleets {
		if newest == nil || fleets[i].CreationTimestamp.After(newest.CreationTimestamp.Time) {
			newest = &fleets[i]
		}
	}
	return newest
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
