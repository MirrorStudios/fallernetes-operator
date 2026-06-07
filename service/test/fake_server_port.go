package test

import (
	"context"

	"github.com/MirrorStudios/fallernetes-service/internal/gen"
)

type FakeServerPort struct {
	CreateServerCalled    bool
	CreateServerName      string
	CreateServerNamespace string
	CreateServerLabels    map[string]string
	CreateServerSpec      gen.ServerSpec
	CreateServerErr       error

	DeleteServerCalled    bool
	DeleteServerName      string
	DeleteServerNamespace string
	DeleteServerForce     bool
	DeleteServerErr       error
}

func (f *FakeServerPort) CreateServer(ctx context.Context, name, namespace string, labels map[string]string, spec gen.ServerSpec) error {
	f.CreateServerCalled = true
	f.CreateServerName = name
	f.CreateServerNamespace = namespace
	f.CreateServerLabels = labels
	f.CreateServerSpec = spec
	return f.CreateServerErr
}

func (f *FakeServerPort) DeleteServer(ctx context.Context, name, namespace string, force bool) error {
	f.DeleteServerCalled = true
	f.DeleteServerName = name
	f.DeleteServerNamespace = namespace
	f.DeleteServerForce = force
	return f.DeleteServerErr
}
