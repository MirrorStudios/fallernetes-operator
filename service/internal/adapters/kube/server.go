package kube

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/MirrorStudios/fallernetes-service/internal/gen"
)

func (k *Adapter) CreateServer(ctx context.Context, name, namespace string, labels map[string]string, spec gen.ServerSpec) error {
	obj := serverCRD{
		APIVersion: crdGroup + "/" + crdVersion,
		Kind:       "Server",
		Metadata:   crdMetadata{Name: name, Namespace: namespace, Labels: labels},
		Spec:       serverSpecToCRD(spec),
	}
	u, err := toUnstructured(obj)
	if err != nil {
		return err
	}
	_, err = k.dynamicClient.Resource(ServerGVR).Namespace(namespace).Create(ctx, u, metav1.CreateOptions{})
	return err
}

func (k *Adapter) DeleteServer(ctx context.Context, name, namespace string, force bool) error {
	err := k.dynamicClient.Resource(ServerGVR).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	if force {
		return k.sendDeleteAllowed(ctx, name, namespace)
	}
	return nil
}

func (k *Adapter) sendDeleteAllowed(ctx context.Context, name, namespace string) error {
	pod, err := k.clientSet.CoreV1().Pods(namespace).Get(ctx, name+"-pod", metav1.GetOptions{})
	if err != nil {
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
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	return nil
}
