package kube

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/MirrorStudios/fallernetes-service/internal/gen"
)

func (k *Adapter) CreateScaler(ctx context.Context, name, namespace string, labels map[string]string, spec gen.GameAutoscalerSpec) error {
	obj := scalerCRD{
		APIVersion: crdGroup + "/" + crdVersion,
		Kind:       "GameAutoscaler",
		Metadata:   crdMetadata{Name: name, Namespace: namespace, Labels: labels},
		Spec:       spec,
	}
	u, err := toUnstructured(obj)
	if err != nil {
		return err
	}
	_, err = k.dynamicClient.Resource(ScalerGVR).Namespace(namespace).Create(ctx, u, metav1.CreateOptions{})
	return err
}

func (k *Adapter) DeleteScaler(ctx context.Context, name, namespace string) error {
	return k.dynamicClient.Resource(ScalerGVR).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}
