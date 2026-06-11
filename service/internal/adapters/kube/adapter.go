package kube

import "sigs.k8s.io/controller-runtime/pkg/client"

type Adapter struct {
	client client.Client
}

func NewKubeAdapter(c client.Client) *Adapter {
	return &Adapter{client: c}
}
