package test

import (
	"context"
	"errors"
	"testing"

	"github.com/MirrorStudios/fallernetes-service/internal/gen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validServerRequest() gen.CreateServerRequestObject {
	return gen.CreateServerRequestObject{
		Body: &gen.CreateServerRequest{
			Name:      "test-server",
			Namespace: "default",
			Spec: gen.ServerSpec{
				Containers: []gen.Container{
					{Name: "game", Image: "registry.example.com/game:latest"},
				},
			},
		},
	}
}

func TestCreateServer_Success(t *testing.T) {
	fake := &FakeServerPort{}
	svc := newTestService(fake, &FakeFleetPort{}, &FakeGamePort{}, &FakeScalerPort{}, &FakePodPort{})

	resp, err := svc.CreateServer(context.Background(), validServerRequest())

	require.NoError(t, err)
	assert.IsType(t, gen.CreateServer201JSONResponse{}, resp)
	assert.True(t, fake.CreateServerCalled)
	assert.Equal(t, "test-server", fake.CreateServerName)
	assert.Equal(t, "default", fake.CreateServerNamespace)
}

func TestCreateServer_KubeError(t *testing.T) {
	fake := &FakeServerPort{CreateServerErr: errors.New("kube unavailable")}
	svc := newTestService(fake, &FakeFleetPort{}, &FakeGamePort{}, &FakeScalerPort{}, &FakePodPort{})

	resp, err := svc.CreateServer(context.Background(), validServerRequest())

	require.NoError(t, err)
	assert.IsType(t, gen.CreateServer500JSONResponse{}, resp)
}

func TestDeleteServer_Success(t *testing.T) {
	fake := &FakeServerPort{}
	svc := newTestService(fake, &FakeFleetPort{}, &FakeGamePort{}, &FakeScalerPort{}, &FakePodPort{})

	force := true
	resp, err := svc.DeleteServer(context.Background(), gen.DeleteServerRequestObject{
		Body: &gen.DeleteObjectRequest{Name: "test-server", Namespace: "default", Force: &force},
	})

	require.NoError(t, err)
	assert.IsType(t, gen.DeleteServer204Response{}, resp)
	assert.True(t, fake.DeleteServerCalled)
	assert.True(t, fake.DeleteServerForce)
}

func TestDeleteServer_KubeError(t *testing.T) {
	fake := &FakeServerPort{DeleteServerErr: errors.New("not found")}
	svc := newTestService(fake, &FakeFleetPort{}, &FakeGamePort{}, &FakeScalerPort{}, &FakePodPort{})

	resp, err := svc.DeleteServer(context.Background(), gen.DeleteServerRequestObject{
		Body: &gen.DeleteObjectRequest{Name: "test-server", Namespace: "default"},
	})

	require.NoError(t, err)
	assert.IsType(t, gen.DeleteServer500JSONResponse{}, resp)
}
