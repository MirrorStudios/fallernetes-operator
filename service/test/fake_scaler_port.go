package test

import (
	"context"

	"github.com/MirrorStudios/fallernetes-service/internal/gen"
)

type FakeScalerPort struct {
	CreateScalerCalled    bool
	CreateScalerName      string
	CreateScalerNamespace string
	CreateScalerSpec      gen.GameAutoscalerSpec
	CreateScalerErr       error

	DeleteScalerCalled    bool
	DeleteScalerName      string
	DeleteScalerNamespace string
	DeleteScalerErr       error
}

func (f *FakeScalerPort) CreateScaler(ctx context.Context, name, namespace string, labels map[string]string, spec gen.GameAutoscalerSpec) error {
	f.CreateScalerCalled = true
	f.CreateScalerName = name
	f.CreateScalerNamespace = namespace
	f.CreateScalerSpec = spec
	return f.CreateScalerErr
}

func (f *FakeScalerPort) DeleteScaler(ctx context.Context, name, namespace string) error {
	f.DeleteScalerCalled = true
	f.DeleteScalerName = name
	f.DeleteScalerNamespace = namespace
	return f.DeleteScalerErr
}
