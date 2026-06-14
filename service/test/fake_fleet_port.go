package test

import (
	"context"

	"github.com/MirrorStudios/fallernetes-operator/service/internal/gen"
)

type FakeFleetPort struct {
	CreateFleetCalled    bool
	CreateFleetName      string
	CreateFleetNamespace string
	CreateFleetSpec      gen.FleetSpec
	CreateFleetErr       error

	DeleteFleetCalled    bool
	DeleteFleetName      string
	DeleteFleetNamespace string
	DeleteFleetForce     bool
	DeleteFleetErr       error

	PatchFleetReplicasCalled    bool
	PatchFleetReplicasName      string
	PatchFleetReplicasNamespace string
	PatchFleetReplicasReplicas  int32
	PatchFleetReplicasErr       error
}

func (f *FakeFleetPort) CreateFleet(_ context.Context, name, namespace string, _ map[string]string, spec gen.FleetSpec) error {
	f.CreateFleetCalled = true
	f.CreateFleetName = name
	f.CreateFleetNamespace = namespace
	f.CreateFleetSpec = spec
	return f.CreateFleetErr
}

func (f *FakeFleetPort) DeleteFleet(_ context.Context, name, namespace string, force bool) error {
	f.DeleteFleetCalled = true
	f.DeleteFleetName = name
	f.DeleteFleetNamespace = namespace
	f.DeleteFleetForce = force
	return f.DeleteFleetErr
}

func (f *FakeFleetPort) PatchFleetReplicas(_ context.Context, name, namespace string, replicas int32) error {
	f.PatchFleetReplicasCalled = true
	f.PatchFleetReplicasName = name
	f.PatchFleetReplicasNamespace = namespace
	f.PatchFleetReplicasReplicas = replicas
	return f.PatchFleetReplicasErr
}
