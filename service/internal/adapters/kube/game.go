package kube

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/MirrorStudios/fallernetes-service/internal/gen"
)

func (k *Adapter) CreateGame(ctx context.Context, name, namespace string, labels map[string]string, spec gen.GameTypeSpec) error {
	obj := gameCRD{
		APIVersion: crdGroup + "/" + crdVersion,
		Kind:       "GameType",
		Metadata:   crdMetadata{Name: name, Namespace: namespace, Labels: labels},
		Spec: gameCRDSpec{
			FleetSpec: fleetSpecToCRD(spec.FleetSpec),
		},
	}
	u, err := toUnstructured(obj)
	if err != nil {
		return err
	}
	_, err = k.dynamicClient.Resource(GameGVR).Namespace(namespace).Create(ctx, u, metav1.CreateOptions{})
	return err
}

func (k *Adapter) DeleteGame(ctx context.Context, name, namespace string, force bool) error {
	err := k.dynamicClient.Resource(GameGVR).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	if force {
		return k.removeFleetsForGame(ctx, name, namespace, force)
	}
	return nil
}

func (k *Adapter) removeFleetsForGame(ctx context.Context, gameName, namespace string, force bool) error {
	fleets, err := k.dynamicClient.Resource(FleetGVR).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, fleet := range fleets.Items {
		if fleet.GetLabels()["type"] != gameName {
			continue
		}
		if err := k.DeleteFleet(ctx, fleet.GetName(), fleet.GetNamespace(), force); err != nil {
			return err
		}
	}
	return nil
}
