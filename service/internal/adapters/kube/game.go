package kube

import (
	"context"

	v1alpha1 "github.com/MirrorStudios/fallernetes-operator/operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/MirrorStudios/fallernetes-operator/service/internal/gen"
)

func (k *Adapter) CreateGame(ctx context.Context, name, namespace string, labels map[string]string, spec gen.GameTypeSpec) error {
	game := &v1alpha1.GameType{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: labels},
		Spec:       gameTypeSpecToOperator(spec),
	}
	return k.client.Create(ctx, game)
}

func (k *Adapter) DeleteGame(ctx context.Context, name, namespace string, force bool) error {
	game := &v1alpha1.GameType{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
	if err := k.client.Delete(ctx, game); err != nil {
		return err
	}
	if force {
		return k.removeFleetsForGame(ctx, name, namespace)
	}
	return nil
}

func (k *Adapter) PatchGameReplicas(ctx context.Context, name, namespace string, replicas int32) error {
	game := &v1alpha1.GameType{}
	if err := k.client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, game); err != nil {
		return err
	}
	game.Spec.FleetSpec.Scaling.Replicas = replicas
	return k.client.Update(ctx, game)
}

func (k *Adapter) removeFleetsForGame(ctx context.Context, gameName, namespace string) error {
	fleetList := &v1alpha1.FleetList{}
	if err := k.client.List(ctx, fleetList, client.InNamespace(namespace), client.MatchingLabels{"type": gameName}); err != nil {
		return err
	}
	for _, fleet := range fleetList.Items {
		if err := k.DeleteFleet(ctx, fleet.Name, fleet.Namespace, true); err != nil {
			return err
		}
	}
	return nil
}
