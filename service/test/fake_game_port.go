package test

import (
	"context"

	"github.com/MirrorStudios/fallernetes-operator/service/internal/gen"
)

type FakeGamePort struct {
	CreateGameCalled    bool
	CreateGameName      string
	CreateGameNamespace string
	CreateGameSpec      gen.GameTypeSpec
	CreateGameErr       error

	DeleteGameCalled    bool
	DeleteGameName      string
	DeleteGameNamespace string
	DeleteGameForce     bool
	DeleteGameErr       error

	PatchGameReplicasCalled    bool
	PatchGameReplicasName      string
	PatchGameReplicasNamespace string
	PatchGameReplicasReplicas  int32
	PatchGameReplicasErr       error
}

func (f *FakeGamePort) CreateGame(_ context.Context, name, namespace string, _ map[string]string, spec gen.GameTypeSpec) error {
	f.CreateGameCalled = true
	f.CreateGameName = name
	f.CreateGameNamespace = namespace
	f.CreateGameSpec = spec
	return f.CreateGameErr
}

func (f *FakeGamePort) DeleteGame(_ context.Context, name, namespace string, force bool) error {
	f.DeleteGameCalled = true
	f.DeleteGameName = name
	f.DeleteGameNamespace = namespace
	f.DeleteGameForce = force
	return f.DeleteGameErr
}

func (f *FakeGamePort) PatchGameReplicas(_ context.Context, name, namespace string, replicas int32) error {
	f.PatchGameReplicasCalled = true
	f.PatchGameReplicasName = name
	f.PatchGameReplicasNamespace = namespace
	f.PatchGameReplicasReplicas = replicas
	return f.PatchGameReplicasErr
}
