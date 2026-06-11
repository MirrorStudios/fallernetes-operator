package test

import (
	"context"
	"testing"

	"github.com/MirrorStudios/fallernetes-operator/service/internal/gen"
	"github.com/MirrorStudios/fallernetes-operator/service/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestService(server *FakeServerPort, fleet *FakeFleetPort, game *FakeGamePort, scaler *FakeScalerPort, pod *FakePodPort) *service.OperatorService {
	return service.NewOperatorService(server, fleet, game, scaler, pod)
}

func TestHealth_ReturnsOK(t *testing.T) {
	svc := newTestService(&FakeServerPort{}, &FakeFleetPort{}, &FakeGamePort{}, &FakeScalerPort{}, &FakePodPort{})
	resp, err := svc.Health(context.Background(), gen.HealthRequestObject{})
	require.NoError(t, err)
	assert.IsType(t, gen.Health200Response{}, resp)
}
