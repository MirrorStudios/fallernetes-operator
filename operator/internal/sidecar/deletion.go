package sidecar

import (
	"context"
	"strconv"
	"time"

	"github.com/MirrorStudios/fallernetes-operator/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Deletion interface {
	IsDeletionAllowed(*v1alpha1.Server, *corev1.Pod) (bool, error)
}

// FleetDeletionChecker checks whether a specific server within a fleet is safe to delete.
type FleetDeletionChecker interface {
	IsDeleteAllowed(ctx context.Context, server *v1alpha1.Server, c *client.Client) (bool, error)
}

type ProdDeletionChecker struct{}

func (p ProdDeletionChecker) IsDeletionAllowed(server *v1alpha1.Server, pod *corev1.Pod) (bool, error) {
	if pod.Status.Phase != corev1.PodRunning {
		return true, nil
	}
	if server.Spec.AllowForceDelete {
		return true, nil
	}

	if server.Spec.TimeOut != nil {
		timeWhenAllowDelete := server.GetDeletionTimestamp().Time.Add(server.Spec.TimeOut.Duration)
		if timeWhenAllowDelete.Before(time.Now()) {
			return true, nil
		}
	}
	port := strconv.Itoa(*server.Spec.SidecarSettings.Port)
	err := RequestShutdown(pod, port)
	if err != nil {
		return false, err
	}
	allowed, err := IsDeleteAllowed(pod, port)
	return allowed, err
}

func (ProdDeletionChecker) IsDeleteAllowed(ctx context.Context, server *v1alpha1.Server, c *client.Client) (bool, error) {
	podName := server.Name + "-pod"
	pod := &corev1.Pod{}
	err := (*c).Get(ctx, types.NamespacedName{Namespace: server.Namespace, Name: podName}, pod)
	if err != nil {
		return false, err
	}

	port := strconv.Itoa(*server.Spec.SidecarSettings.Port)
	allowed, err := IsDeleteAllowed(pod, port)
	if err != nil {
		return false, nil
	}

	return allowed, nil
}
