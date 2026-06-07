package kube

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/MirrorStudios/fallernetes-service/internal/gen"
)

const (
	crdGroup   = "gameserver.falloria.com"
	crdVersion = "v1alpha1"
)

var (
	ServerGVR = schema.GroupVersionResource{Group: crdGroup, Version: crdVersion, Resource: "servers"}
	FleetGVR  = schema.GroupVersionResource{Group: crdGroup, Version: crdVersion, Resource: "fleets"}
	GameGVR   = schema.GroupVersionResource{Group: crdGroup, Version: crdVersion, Resource: "gametypes"}
	ScalerGVR = schema.GroupVersionResource{Group: crdGroup, Version: crdVersion, Resource: "gameautoscalers"}
)

type crdMetadata struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Labels    map[string]string `json:"labels,omitempty"`
}

type serverCRDSpec struct {
	Pod              v1.PodSpec           `json:"pod,omitempty"`
	TimeOut          *string              `json:"timeout,omitempty"`
	AllowForceDelete bool                 `json:"allowForceDelete,omitempty"`
	Sidecar          *gen.SidecarSettings `json:"sidecar,omitempty"`
	GameInfo         *gen.GameInfo        `json:"gameInfo,omitempty"`
}

type serverCRD struct {
	APIVersion string        `json:"apiVersion"`
	Kind       string        `json:"kind"`
	Metadata   crdMetadata   `json:"metadata"`
	Spec       serverCRDSpec `json:"spec"`
}

type fleetCRDSpec struct {
	Spec    serverCRDSpec   `json:"spec"`
	Scaling fleetCRDScaling `json:"scaling"`
}

type fleetCRDScaling struct {
	Replicas          int32  `json:"replicas"`
	PrioritizeAllowed bool   `json:"prioritizeAllowed,omitempty"`
	AgePriority       string `json:"agePriority,omitempty"`
}

type fleetCRD struct {
	APIVersion string       `json:"apiVersion"`
	Kind       string       `json:"kind"`
	Metadata   crdMetadata  `json:"metadata"`
	Spec       fleetCRDSpec `json:"spec"`
}

type gameCRDSpec struct {
	FleetSpec fleetCRDSpec `json:"fleetSpec"`
}

type gameCRD struct {
	APIVersion string      `json:"apiVersion"`
	Kind       string      `json:"kind"`
	Metadata   crdMetadata `json:"metadata"`
	Spec       gameCRDSpec `json:"spec"`
}

type scalerCRD struct {
	APIVersion string                 `json:"apiVersion"`
	Kind       string                 `json:"kind"`
	Metadata   crdMetadata            `json:"metadata"`
	Spec       gen.GameAutoscalerSpec `json:"spec"`
}
