package kube

import (
	"time"

	v1alpha1 "github.com/MirrorStudios/fallernetes-operator/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/MirrorStudios/fallernetes-operator/service/internal/gen"
)

func serverSpecToOperator(spec gen.ServerSpec) v1alpha1.ServerSpec {
	containers := make([]corev1.Container, len(spec.Containers))
	for i, c := range spec.Containers {
		containers[i] = containerToKube(c)
	}
	podSpec := corev1.PodSpec{Containers: containers}
	if spec.InitContainers != nil {
		inits := make([]corev1.Container, len(*spec.InitContainers))
		for i, c := range *spec.InitContainers {
			inits[i] = containerToKube(c)
		}
		podSpec.InitContainers = inits
	}

	s := v1alpha1.ServerSpec{Pod: podSpec}
	if spec.AllowForceDelete != nil {
		s.AllowForceDelete = *spec.AllowForceDelete
	}
	if spec.Timeout != nil {
		if d, err := time.ParseDuration(*spec.Timeout); err == nil {
			s.TimeOut = &metav1.Duration{Duration: d}
		}
	}
	if spec.Sidecar != nil {
		s.SidecarSettings = sidecarToOperator(spec.Sidecar)
	}
	if spec.GameInfo != nil {
		s.GameInfo = gameInfoToOperator(spec.GameInfo)
	}
	return s
}

func sidecarToOperator(s *gen.SidecarSettings) *v1alpha1.SidecarSettings {
	out := &v1alpha1.SidecarSettings{SidecarImage: &s.Image}
	if s.Port != nil {
		port := int(*s.Port)
		out.Port = &port
	}
	if s.LogDebug != nil {
		out.LogDebug = *s.LogDebug
	}
	return out
}

func gameInfoToOperator(g *gen.GameInfo) *v1alpha1.GameInfo {
	if g.Capacity == nil {
		return &v1alpha1.GameInfo{}
	}
	cap := int(*g.Capacity)
	return &v1alpha1.GameInfo{Capacity: &cap}
}

func fleetSpecToOperator(spec gen.FleetSpec) v1alpha1.FleetSpec {
	prioritizeAllowed := false
	if spec.Scaling.PrioritizeAllowed != nil {
		prioritizeAllowed = *spec.Scaling.PrioritizeAllowed
	}
	agePriority := v1alpha1.OldestFirst
	if spec.Scaling.AgePriority != nil {
		agePriority = v1alpha1.Priority(*spec.Scaling.AgePriority)
	}
	return v1alpha1.FleetSpec{
		ServerSpec: serverSpecToOperator(spec.ServerSpec),
		Scaling: v1alpha1.FleetScaling{
			Replicas:          spec.Scaling.Replicas,
			PrioritizeAllowed: prioritizeAllowed,
			AgePriority:       agePriority,
		},
	}
}

func gameTypeSpecToOperator(spec gen.GameTypeSpec) v1alpha1.GameTypeSpec {
	return v1alpha1.GameTypeSpec{FleetSpec: fleetSpecToOperator(spec.FleetSpec)}
}

func scalerSpecToOperator(spec gen.GameAutoscalerSpec) v1alpha1.GameTypeAutoscalerSpec {
	webhookSpec := v1alpha1.WebhookAutoscalerSpec{
		Url:  spec.Policy.Webhook.Url,
		Path: &spec.Policy.Webhook.Path,
		Service: &v1alpha1.Service{
			Name:      spec.Policy.Webhook.Service.Name,
			Namespace: spec.Policy.Webhook.Service.Namespace,
			Port:      int(spec.Policy.Webhook.Service.Port),
		},
	}
	return v1alpha1.GameTypeAutoscalerSpec{
		GameTypeName: spec.GameTypeName,
		AutoscalePolicy: v1alpha1.AutoscalePolicy{
			Type:                  v1alpha1.Webhook,
			WebhookAutoscalerSpec: webhookSpec,
		},
		Sync: v1alpha1.Sync{
			Type: v1alpha1.FixedInterval,
			Time: &metav1.Duration{Duration: time.Duration(spec.Sync.FixedInterval) * time.Second},
		},
	}
}

func containerToKube(c gen.Container) corev1.Container {
	kc := corev1.Container{
		Name:  c.Name,
		Image: c.Image,
	}
	if c.Env != nil {
		for _, e := range *c.Env {
			kc.Env = append(kc.Env, corev1.EnvVar{Name: e.Name, Value: e.Value})
		}
	}
	if c.Ports != nil {
		for _, p := range *c.Ports {
			cp := corev1.ContainerPort{ContainerPort: p.Port}
			if p.Name != nil {
				cp.Name = *p.Name
			}
			if p.Protocol == gen.UDP {
				cp.Protocol = corev1.ProtocolUDP
			} else {
				cp.Protocol = corev1.ProtocolTCP
			}
			kc.Ports = append(kc.Ports, cp)
		}
	}
	if c.Resources != nil {
		kc.Resources.Requests = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(c.Resources.CpuRequest),
			corev1.ResourceMemory: resource.MustParse(c.Resources.MemoryRequest),
		}
		if c.Resources.CpuLimit != nil || c.Resources.MemoryLimit != nil {
			kc.Resources.Limits = corev1.ResourceList{}
			if c.Resources.CpuLimit != nil {
				kc.Resources.Limits[corev1.ResourceCPU] = resource.MustParse(*c.Resources.CpuLimit)
			}
			if c.Resources.MemoryLimit != nil {
				kc.Resources.Limits[corev1.ResourceMemory] = resource.MustParse(*c.Resources.MemoryLimit)
			}
		}
	}
	return kc
}
