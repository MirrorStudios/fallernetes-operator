package kubeport

import "context"

type PodPort interface {
	AddPodLabel(ctx context.Context, serverName, namespace, key, value string) error
	RemovePodLabel(ctx context.Context, serverName, namespace, key string) error
}
