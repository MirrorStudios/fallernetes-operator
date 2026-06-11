package test

import (
	"context"
	"testing"

	"github.com/MirrorStudios/fallernetes-operator/sidecar/internal/gen"
	"github.com/MirrorStudios/fallernetes-operator/sidecar/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestService(state *FakeStatePort) *service.SidecarService {
	return service.NewSidecarService(state)
}

func TestHealth_ReturnsOK(t *testing.T) {
	svc := newTestService(&FakeStatePort{})
	resp, err := svc.Health(context.Background(), gen.HealthRequestObject{})
	require.NoError(t, err)
	assert.IsType(t, gen.Health200Response{}, resp)
}
