package utils

import (
	"context"

	"github.com/MirrorStudios/fallernetes-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func CreateServerForFleet(fleet v1alpha1.Fleet, namespace string) *v1alpha1.Server {
	labels := fleet.Labels
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["fleet"] = fleet.Name
	server := v1alpha1.Server{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fleet.Name + "-",
			Namespace:    namespace,
			Labels:       labels,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(&fleet, v1alpha1.GroupVersion.WithKind("Fleet")),
			},
		},
		Spec: fleet.Spec.ServerSpec,
	}

	return &server
}
