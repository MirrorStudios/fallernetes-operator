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
}

func (f *FakeGamePort) CreateGame(ctx context.Context, name, namespace string, labels map[string]string, spec gen.GameTypeSpec) error {
	f.CreateGameCalled = true
	f.CreateGameName = name
	f.CreateGameNamespace = namespace
	f.CreateGameSpec = spec
	return f.CreateGameErr
}

func (f *FakeGamePort) DeleteGame(ctx context.Context, name, namespace string, force bool) error {
	f.DeleteGameCalled = true
	f.DeleteGameName = name
	f.DeleteGameNamespace = namespace
	f.DeleteGameForce = force
	return f.DeleteGameErr
}
