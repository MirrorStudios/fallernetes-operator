package service

import (
	kubeport "github.com/MirrorStudios/fallernetes-service/internal/ports/kube"
)

type OperatorService struct {
	server kubeport.ServerPort
	fleet  kubeport.FleetPort
	game   kubeport.GamePort
	scaler kubeport.ScalerPort
	pod    kubeport.PodPort
}

func NewOperatorService(
	server kubeport.ServerPort,
	fleet kubeport.FleetPort,
	game kubeport.GamePort,
	scaler kubeport.ScalerPort,
	pod kubeport.PodPort,
) *OperatorService {
	return &OperatorService{
		server: server,
		fleet:  fleet,
		game:   game,
		scaler: scaler,
		pod:    pod,
	}
}

func strPtr(s string) *string {
	return &s
}
