package kube

import (
	"encoding/json"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/MirrorStudios/fallernetes-service/internal/gen"
)

func toUnstructured(obj any) (*unstructured.Unstructured, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	u := &unstructured.Unstructured{}
	if err := u.UnmarshalJSON(data); err != nil {
		return nil, err
	}
	return u, nil
}

func serverSpecToCRD(spec gen.ServerSpec) serverCRDSpec {
	containers := make([]v1.Container, len(spec.Containers))
	for i, c := range spec.Containers {
		containers[i] = containerToKube(c)
	}

	podSpec := v1.PodSpec{Containers: containers}
	if spec.InitContainers != nil {
		inits := make([]v1.Container, len(*spec.InitContainers))
		for i, c := range *spec.InitContainers {
			inits[i] = containerToKube(c)
		}
		podSpec.InitContainers = inits
	}

	crd := serverCRDSpec{
		Pod:      podSpec,
		Sidecar:  spec.Sidecar,
		GameInfo: spec.GameInfo,
	}
	if spec.Timeout != nil {
		crd.TimeOut = spec.Timeout
	}
	if spec.AllowForceDelete != nil {
		crd.AllowForceDelete = *spec.AllowForceDelete
	}
	return crd
}

func fleetSpecToCRD(spec gen.FleetSpec) fleetCRDSpec {
	agePriority := ""
	if spec.Scaling.AgePriority != nil {
		agePriority = string(*spec.Scaling.AgePriority)
	}
	prioritizeAllowed := false
	if spec.Scaling.PrioritizeAllowed != nil {
		prioritizeAllowed = *spec.Scaling.PrioritizeAllowed
	}
	return fleetCRDSpec{
		Spec: serverSpecToCRD(spec.ServerSpec),
		Scaling: fleetCRDScaling{
			Replicas:          spec.Scaling.Replicas,
			PrioritizeAllowed: prioritizeAllowed,
			AgePriority:       agePriority,
		},
	}
}

func containerToKube(c gen.Container) v1.Container {
	kc := v1.Container{
		Name:  c.Name,
		Image: c.Image,
	}
	if c.Env != nil {
		for _, e := range *c.Env {
			kc.Env = append(kc.Env, v1.EnvVar{Name: e.Name, Value: e.Value})
		}
	}
	if c.Ports != nil {
		for _, p := range *c.Ports {
			cp := v1.ContainerPort{ContainerPort: p.Port}
			if p.Name != nil {
				cp.Name = *p.Name
			}
			if p.Protocol == gen.UDP {
				cp.Protocol = v1.ProtocolUDP
			} else {
				cp.Protocol = v1.ProtocolTCP
			}
			kc.Ports = append(kc.Ports, cp)
		}
	}
	if c.Resources != nil {
		kc.Resources.Requests = v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse(c.Resources.CpuRequest),
			v1.ResourceMemory: resource.MustParse(c.Resources.MemoryRequest),
		}
		if c.Resources.CpuLimit != nil || c.Resources.MemoryLimit != nil {
			kc.Resources.Limits = v1.ResourceList{}
			if c.Resources.CpuLimit != nil {
				kc.Resources.Limits[v1.ResourceCPU] = resource.MustParse(*c.Resources.CpuLimit)
			}
			if c.Resources.MemoryLimit != nil {
				kc.Resources.Limits[v1.ResourceMemory] = resource.MustParse(*c.Resources.MemoryLimit)
			}
		}
	}
	return kc
}
