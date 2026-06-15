package test

import (
	"context"
	"errors"
	"testing"

	"github.com/MirrorStudios/fallernetes-operator/service/internal/gen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validFleetRequest() gen.CreateFleetRequestObject {
	replicas := int32(3)
	return gen.CreateFleetRequestObject{
		Body: &gen.CreateFleetRequest{
			Name:      "test-fleet",
			Namespace: "default",
			Spec: gen.FleetSpec{
				ServerSpec: gen.ServerSpec{
					Containers: []gen.Container{
						{Name: "game", Image: "registry.example.com/game:latest"},
					},
				},
				Scaling: gen.FleetScaling{Replicas: replicas},
			},
		},
	}
}

func TestCreateFleet_Success(t *testing.T) {
	fake := &FakeFleetPort{}
	svc := newTestService(&FakeServerPort{}, fake, &FakeGamePort{}, &FakeScalerPort{}, &FakePodPort{})

	resp, err := svc.CreateFleet(context.Background(), validFleetRequest())

	require.NoError(t, err)
	assert.IsType(t, gen.CreateFleet201JSONResponse{}, resp)
	assert.True(t, fake.CreateFleetCalled)
	assert.Equal(t, "test-fleet", fake.CreateFleetName)
}

func TestCreateFleet_KubeError(t *testing.T) {
	fake := &FakeFleetPort{CreateFleetErr: errors.New("kube unavailable")}
	svc := newTestService(&FakeServerPort{}, fake, &FakeGamePort{}, &FakeScalerPort{}, &FakePodPort{})

	resp, err := svc.CreateFleet(context.Background(), validFleetRequest())

	require.NoError(t, err)
	assert.IsType(t, gen.CreateFleet500JSONResponse{}, resp)
}

func TestDeleteFleet_Success(t *testing.T) {
	fake := &FakeFleetPort{}
	svc := newTestService(&FakeServerPort{}, fake, &FakeGamePort{}, &FakeScalerPort{}, &FakePodPort{})

	resp, err := svc.DeleteFleet(context.Background(), gen.DeleteFleetRequestObject{
		Body: &gen.DeleteObjectRequest{Name: "test-fleet", Namespace: "default"},
	})

	require.NoError(t, err)
	assert.IsType(t, gen.DeleteFleet204Response{}, resp)
	assert.True(t, fake.DeleteFleetCalled)
}

func TestDeleteFleet_KubeError(t *testing.T) {
	fake := &FakeFleetPort{DeleteFleetErr: errors.New("not found")}
	svc := newTestService(&FakeServerPort{}, fake, &FakeGamePort{}, &FakeScalerPort{}, &FakePodPort{})

	resp, err := svc.DeleteFleet(context.Background(), gen.DeleteFleetRequestObject{
		Body: &gen.DeleteObjectRequest{Name: "test-fleet", Namespace: "default"},
	})

	require.NoError(t, err)
	assert.IsType(t, gen.DeleteFleet500JSONResponse{}, resp)
}

func TestPatchFleetReplicas_Success(t *testing.T) {
	fake := &FakeFleetPort{}
	svc := newTestService(&FakeServerPort{}, fake, &FakeGamePort{}, &FakeScalerPort{}, &FakePodPort{})

	replicas := int32(5)
	resp, err := svc.PatchFleetReplicas(context.Background(), gen.PatchFleetReplicasRequestObject{
		Body: &gen.PatchReplicasRequest{Name: "test-fleet", Namespace: "default", Replicas: replicas},
	})

	require.NoError(t, err)
	assert.IsType(t, gen.PatchFleetReplicas200Response{}, resp)
	assert.True(t, fake.PatchFleetReplicasCalled)
	assert.Equal(t, "test-fleet", fake.PatchFleetReplicasName)
	assert.Equal(t, int32(5), fake.PatchFleetReplicasReplicas)
}

func TestPatchFleetReplicas_KubeError(t *testing.T) {
	fake := &FakeFleetPort{PatchFleetReplicasErr: errors.New("not found")}
	svc := newTestService(&FakeServerPort{}, fake, &FakeGamePort{}, &FakeScalerPort{}, &FakePodPort{})

	replicas := int32(5)
	resp, err := svc.PatchFleetReplicas(context.Background(), gen.PatchFleetReplicasRequestObject{
		Body: &gen.PatchReplicasRequest{Name: "test-fleet", Namespace: "default", Replicas: replicas},
	})

	require.NoError(t, err)
	assert.IsType(t, gen.PatchFleetReplicas500JSONResponse{}, resp)
}
