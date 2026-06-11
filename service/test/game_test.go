package test

import (
	"context"
	"errors"
	"testing"

	"github.com/MirrorStudios/fallernetes-operator/service/internal/gen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validGameRequest() gen.CreateGameRequestObject {
	replicas := int32(2)
	return gen.CreateGameRequestObject{
		Body: &gen.CreateGameRequest{
			Name:      "test-game",
			Namespace: "default",
			Spec: gen.GameTypeSpec{
				FleetSpec: gen.FleetSpec{
					ServerSpec: gen.ServerSpec{
						Containers: []gen.Container{
							{Name: "game", Image: "registry.example.com/game:latest"},
						},
					},
					Scaling: gen.FleetScaling{Replicas: replicas},
				},
			},
		},
	}
}

func TestCreateGame_Success(t *testing.T) {
	fake := &FakeGamePort{}
	svc := newTestService(&FakeServerPort{}, &FakeFleetPort{}, fake, &FakeScalerPort{}, &FakePodPort{})

	resp, err := svc.CreateGame(context.Background(), validGameRequest())

	require.NoError(t, err)
	assert.IsType(t, gen.CreateGame201JSONResponse{}, resp)
	assert.True(t, fake.CreateGameCalled)
	assert.Equal(t, "test-game", fake.CreateGameName)
}

func TestCreateGame_KubeError(t *testing.T) {
	fake := &FakeGamePort{CreateGameErr: errors.New("kube unavailable")}
	svc := newTestService(&FakeServerPort{}, &FakeFleetPort{}, fake, &FakeScalerPort{}, &FakePodPort{})

	resp, err := svc.CreateGame(context.Background(), validGameRequest())

	require.NoError(t, err)
	assert.IsType(t, gen.CreateGame500JSONResponse{}, resp)
}

func TestDeleteGame_Success(t *testing.T) {
	fake := &FakeGamePort{}
	svc := newTestService(&FakeServerPort{}, &FakeFleetPort{}, fake, &FakeScalerPort{}, &FakePodPort{})

	resp, err := svc.DeleteGame(context.Background(), gen.DeleteGameRequestObject{
		Body: &gen.DeleteObjectRequest{Name: "test-game", Namespace: "default"},
	})

	require.NoError(t, err)
	assert.IsType(t, gen.DeleteGame204Response{}, resp)
	assert.True(t, fake.DeleteGameCalled)
}

func TestDeleteGame_KubeError(t *testing.T) {
	fake := &FakeGamePort{DeleteGameErr: errors.New("not found")}
	svc := newTestService(&FakeServerPort{}, &FakeFleetPort{}, fake, &FakeScalerPort{}, &FakePodPort{})

	resp, err := svc.DeleteGame(context.Background(), gen.DeleteGameRequestObject{
		Body: &gen.DeleteObjectRequest{Name: "test-game", Namespace: "default"},
	})

	require.NoError(t, err)
	assert.IsType(t, gen.DeleteGame500JSONResponse{}, resp)
}
