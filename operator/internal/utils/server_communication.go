package utils

import (
	"github.com/MirrorStudios/fallernetes/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"strconv"
	"time"
)

type Deletion interface {
	IsDeletionAllowed(*v1alpha1.Server, *corev1.Pod) (bool, error)
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
