package kube

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	v1alpha1 "github.com/MirrorStudios/fallernetes-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/MirrorStudios/fallernetes-service/internal/gen"
)

func (k *Adapter) CreateServer(ctx context.Context, name, namespace string, labels map[string]string, spec gen.ServerSpec) error {
	server := &v1alpha1.Server{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: labels},
		Spec:       serverSpecToOperator(spec),
	}
	return k.client.Create(ctx, server)
}

func (k *Adapter) DeleteServer(ctx context.Context, name, namespace string, force bool) error {
	server := &v1alpha1.Server{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
	if err := k.client.Delete(ctx, server); err != nil {
		return err
	}
	if force {
		return k.sendDeleteAllowed(ctx, name, namespace)
	}
	return nil
}

func (k *Adapter) sendDeleteAllowed(ctx context.Context, name, namespace string) error {
	pod := &corev1.Pod{}
	if err := k.client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name + "-pod"}, pod); err != nil {
		return err
	}
	payload, err := json.Marshal(map[string]any{"allowed": true})
	if err != nil {
		return err
	}
	url := fmt.Sprintf("http://%s:8080/allow_delete", pod.Status.PodIP)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}
