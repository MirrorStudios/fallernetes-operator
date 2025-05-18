package utils

import (
	"github.com/MirrorStudios/fallernetes/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"strconv"
)

func addContainer(spec *corev1.PodSpec, container corev1.Container) *corev1.PodSpec {
	spec.Containers = append(spec.Containers, container)
	return spec
}

func getPodSpec(server *v1alpha1.Server) *corev1.PodSpec {
	spec := server.Spec
	port := spec.SidecarSettings.Port
	portStr := strconv.Itoa(int(*port))
	pod := addContainer(&spec.Pod, corev1.Container{
		Name:  "fallernetes-sidecar",
		Image: spec.SidecarSettings.SidecarImage,
		Ports: []corev1.ContainerPort{
			{
				Name:          "http",
				ContainerPort: 8080,
			},
		},
		Env: []corev1.EnvVar{
			{
				Name:  "SERVER_PORT",
				Value: portStr,
			},
		},
		ImagePullPolicy: corev1.PullIfNotPresent,
	})

	for i := range pod.Containers {
		container := &pod.Containers[i]
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  "CONTAINER_IMAGE",
			Value: container.Image,
		})
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  "SERVER_NAME",
			Value: server.Name,
		})
		if fleet, ok := server.Labels["fleet"]; ok {
			container.Env = append(container.Env, corev1.EnvVar{
				Name:  "FLEET_NAME",
				Value: fleet,
			})
		}

		if fleet, ok := server.Labels["gametype"]; ok {
			container.Env = append(container.Env, corev1.EnvVar{
				Name:  "GAME_NAME",
				Value: fleet,
			})
		}
		container.Env = append(container.Env, corev1.EnvVar{
			Name: "POD_IP",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "status.podIP",
				},
			},
		})
		container.Env = append(container.Env, corev1.EnvVar{
			Name: "NODE_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath:  "spec.nodeName",
					APIVersion: "v1",
				},
			},
		})
	}

	pod.ImagePullSecrets = append(pod.ImagePullSecrets, corev1.LocalObjectReference{
		Name: os.Getenv("IMAGE_PULL_SECRET_NAME"),
	})

	return pod
}

func GetNewPod(server *v1alpha1.Server, namespace string) *corev1.Pod {
	labels := server.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	spec := getPodSpec(server)
	labels["server"] = server.Name
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      server.Name + "-pod",
			Namespace: namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(server, v1alpha1.GroupVersion.WithKind("Server")),
			},
		},
		Spec: *spec,
	}
	return pod
}
