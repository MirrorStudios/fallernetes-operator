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
}

func (f *FakeFleetPort) CreateFleet(ctx context.Context, name, namespace string, labels map[string]string, spec gen.FleetSpec) error {
	f.CreateFleetCalled = true
	f.CreateFleetName = name
	f.CreateFleetNamespace = namespace
	f.CreateFleetSpec = spec
	return f.CreateFleetErr
}

func (f *FakeFleetPort) DeleteFleet(ctx context.Context, name, namespace string, force bool) error {
	f.DeleteFleetCalled = true
	f.DeleteFleetName = name
	f.DeleteFleetNamespace = namespace
	f.DeleteFleetForce = force
	return f.DeleteFleetErr
}
