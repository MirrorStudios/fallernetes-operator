package kube

import (
	"context"
	"maps"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (k *Adapter) AddPodLabel(ctx context.Context, serverName, namespace, key, value string) error {
	pods := k.clientSet.CoreV1().Pods(namespace)
	pod, err := pods.Get(ctx, serverName+"-pod", metav1.GetOptions{})
	if err != nil {
		return err
	}
	labels := pod.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	labels[key] = value
	pod.SetLabels(labels)
	_, err = pods.Update(ctx, pod, metav1.UpdateOptions{})
	return err
}

func (k *Adapter) RemovePodLabel(ctx context.Context, serverName, namespace, key string) error {
	pods := k.clientSet.CoreV1().Pods(namespace)
	pod, err := pods.Get(ctx, serverName+"-pod", metav1.GetOptions{})
	if err != nil {
		return err
	}
	labels := maps.Clone(pod.GetLabels())
	delete(labels, key)
	pod.SetLabels(labels)
	_, err = pods.Update(ctx, pod, metav1.UpdateOptions{})
	return err
}
