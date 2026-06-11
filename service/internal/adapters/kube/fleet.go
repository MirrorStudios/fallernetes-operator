package kube

import (
	"context"

	v1alpha1 "github.com/MirrorStudios/fallernetes-operator/operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/MirrorStudios/fallernetes-operator/service/internal/gen"
)

func (k *Adapter) CreateFleet(ctx context.Context, name, namespace string, labels map[string]string, spec gen.FleetSpec) error {
	fleet := &v1alpha1.Fleet{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: labels},
		Spec:       fleetSpecToOperator(spec),
	}
	return k.client.Create(ctx, fleet)
}

func (k *Adapter) DeleteFleet(ctx context.Context, name, namespace string, force bool) error {
	fleet := &v1alpha1.Fleet{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
	if err := k.client.Delete(ctx, fleet); err != nil {
		return err
	}
	if force {
		return k.forceDeleteFleet(ctx, name, namespace)
	}
	return nil
}

func (k *Adapter) forceDeleteFleet(ctx context.Context, name, namespace string) error {
	serverList := &v1alpha1.ServerList{}
	if err := k.client.List(ctx, serverList, client.InNamespace(namespace), client.MatchingLabels{"fleet": name}); err != nil {
		return err
	}
	for _, server := range serverList.Items {
		if err := k.sendDeleteAllowed(ctx, server.Name, server.Namespace); err != nil {
			return err
		}
	}
	return nil
}
