package builders

import (
	"github.com/MirrorStudios/fallernetes-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
