package kube

import (
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

type Adapter struct {
	dynamicClient *dynamic.DynamicClient
	clientSet     *kubernetes.Clientset
}

func NewKubeAdapter(dynamicClient *dynamic.DynamicClient, clientSet *kubernetes.Clientset) *Adapter {
	return &Adapter{
		dynamicClient: dynamicClient,
		clientSet:     clientSet,
	}
}
