package test

import (
	"context"
	"errors"
	"testing"

	"github.com/MirrorStudios/fallernetes-service/internal/gen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddPodLabel_Success(t *testing.T) {
	fake := &FakePodPort{}
	svc := newTestService(&FakeServerPort{}, &FakeFleetPort{}, &FakeGamePort{}, &FakeScalerPort{}, fake)

	resp, err := svc.AddPodLabel(context.Background(), gen.AddPodLabelRequestObject{
		Body: &gen.AddPodLabelRequest{
			ServerName: "test-server",
			Namespace:  "default",
			Key:        "agones.dev/sdk-node-name",
			Value:      "node-1",
		},
	})

	require.NoError(t, err)
	assert.IsType(t, gen.AddPodLabel200Response{}, resp)
	assert.True(t, fake.AddLabelCalled)
	assert.Equal(t, "test-server", fake.AddLabelServer)
	assert.Equal(t, "agones.dev/sdk-node-name", fake.AddLabelKey)
	assert.Equal(t, "node-1", fake.AddLabelValue)
}

func TestAddPodLabel_KubeError(t *testing.T) {
	fake := &FakePodPort{AddLabelErr: errors.New("pod not found")}
	svc := newTestService(&FakeServerPort{}, &FakeFleetPort{}, &FakeGamePort{}, &FakeScalerPort{}, fake)

	resp, err := svc.AddPodLabel(context.Background(), gen.AddPodLabelRequestObject{
		Body: &gen.AddPodLabelRequest{
			ServerName: "test-server", Namespace: "default", Key: "k", Value: "v",
		},
	})

	require.NoError(t, err)
	assert.IsType(t, gen.AddPodLabel500JSONResponse{}, resp)
}

func TestRemovePodLabel_Success(t *testing.T) {
	fake := &FakePodPort{}
	svc := newTestService(&FakeServerPort{}, &FakeFleetPort{}, &FakeGamePort{}, &FakeScalerPort{}, fake)

	resp, err := svc.RemovePodLabel(context.Background(), gen.RemovePodLabelRequestObject{
		Body: &gen.RemovePodLabelRequest{
			ServerName: "test-server", Namespace: "default", Key: "agones.dev/sdk-node-name",
		},
	})

	require.NoError(t, err)
	assert.IsType(t, gen.RemovePodLabel204Response{}, resp)
	assert.True(t, fake.RemoveLabelCalled)
	assert.Equal(t, "agones.dev/sdk-node-name", fake.RemoveLabelKey)
}

func TestRemovePodLabel_KubeError(t *testing.T) {
	fake := &FakePodPort{RemoveLabelErr: errors.New("pod not found")}
	svc := newTestService(&FakeServerPort{}, &FakeFleetPort{}, &FakeGamePort{}, &FakeScalerPort{}, fake)

	resp, err := svc.RemovePodLabel(context.Background(), gen.RemovePodLabelRequestObject{
		Body: &gen.RemovePodLabelRequest{
			ServerName: "test-server", Namespace: "default", Key: "k",
		},
	})

	require.NoError(t, err)
	assert.IsType(t, gen.RemovePodLabel500JSONResponse{}, resp)
}
