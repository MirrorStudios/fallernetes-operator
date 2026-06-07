package kube

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/MirrorStudios/fallernetes-service/internal/gen"
)

func (k *Adapter) CreateFleet(ctx context.Context, name, namespace string, labels map[string]string, spec gen.FleetSpec) error {
	obj := fleetCRD{
		APIVersion: crdGroup + "/" + crdVersion,
		Kind:       "Fleet",
		Metadata:   crdMetadata{Name: name, Namespace: namespace, Labels: labels},
		Spec:       fleetSpecToCRD(spec),
	}
	u, err := toUnstructured(obj)
	if err != nil {
		return err
	}
	_, err = k.dynamicClient.Resource(FleetGVR).Namespace(namespace).Create(ctx, u, metav1.CreateOptions{})
	return err
}

func (k *Adapter) DeleteFleet(ctx context.Context, name, namespace string, force bool) error {
	err := k.dynamicClient.Resource(FleetGVR).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	if force {
		return k.forceDeleteFleet(ctx, name, namespace)
	}
	return nil
}

func (k *Adapter) forceDeleteFleet(ctx context.Context, name, namespace string) error {
	servers, err := k.dynamicClient.Resource(ServerGVR).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, server := range servers.Items {
		if server.GetLabels()["fleet"] != name {
			continue
		}
		if err := k.sendDeleteAllowed(ctx, server.GetName(), server.GetNamespace()); err != nil {
			return err
		}
	}
	return nil
}
