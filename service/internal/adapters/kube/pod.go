package kube

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (k *Adapter) AddPodLabel(ctx context.Context, serverName, namespace, key, value string) error {
	pod := &corev1.Pod{}
	if err := k.client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: serverName + "-pod"}, pod); err != nil {
		return err
	}
	if pod.Labels == nil {
		pod.Labels = map[string]string{}
	}
	pod.Labels[key] = value
	return k.client.Update(ctx, pod)
}

func (k *Adapter) RemovePodLabel(ctx context.Context, serverName, namespace, key string) error {
	pod := &corev1.Pod{}
	if err := k.client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: serverName + "-pod"}, pod); err != nil {
		return err
	}
	delete(pod.Labels, key)
	return k.client.Update(ctx, pod)
}
