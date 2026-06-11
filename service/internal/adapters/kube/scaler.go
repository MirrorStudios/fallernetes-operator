package kube

import (
	"context"

	v1alpha1 "github.com/MirrorStudios/fallernetes-operator/operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/MirrorStudios/fallernetes-service/internal/gen"
)

func (k *Adapter) CreateScaler(ctx context.Context, name, namespace string, labels map[string]string, spec gen.GameAutoscalerSpec) error {
	scaler := &v1alpha1.GameTypeAutoscaler{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: labels},
		Spec:       scalerSpecToOperator(spec),
	}
	return k.client.Create(ctx, scaler)
}

func (k *Adapter) DeleteScaler(ctx context.Context, name, namespace string) error {
	scaler := &v1alpha1.GameTypeAutoscaler{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
	return k.client.Delete(ctx, scaler)
}
