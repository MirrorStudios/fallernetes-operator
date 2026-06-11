package test

import (
	"context"
	"errors"
	"testing"

	"github.com/MirrorStudios/fallernetes-operator/service/internal/gen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validScalerRequest() gen.CreateScalerRequestObject {
	return gen.CreateScalerRequestObject{
		Body: &gen.CreateScalerRequest{
			Name:      "test-scaler",
			Namespace: "default",
			Spec: gen.GameAutoscalerSpec{
				GameTypeName: "test-game",
				Policy: gen.AutoscalePolicy{
					Type: "webhook",
					Webhook: gen.WebhookSpec{
						Path: "/scale",
						Service: gen.WebhookService{
							Name: "scaler-svc", Namespace: "default", Port: 8080,
						},
					},
				},
				Sync: gen.Sync{Type: "fixedinterval", FixedInterval: 30},
			},
		},
	}
}

func TestCreateScaler_Success(t *testing.T) {
	fake := &FakeScalerPort{}
	svc := newTestService(&FakeServerPort{}, &FakeFleetPort{}, &FakeGamePort{}, fake, &FakePodPort{})

	resp, err := svc.CreateScaler(context.Background(), validScalerRequest())

	require.NoError(t, err)
	assert.IsType(t, gen.CreateScaler201JSONResponse{}, resp)
	assert.True(t, fake.CreateScalerCalled)
	assert.Equal(t, "test-scaler", fake.CreateScalerName)
}

func TestCreateScaler_KubeError(t *testing.T) {
	fake := &FakeScalerPort{CreateScalerErr: errors.New("kube unavailable")}
	svc := newTestService(&FakeServerPort{}, &FakeFleetPort{}, &FakeGamePort{}, fake, &FakePodPort{})

	resp, err := svc.CreateScaler(context.Background(), validScalerRequest())

	require.NoError(t, err)
	assert.IsType(t, gen.CreateScaler500JSONResponse{}, resp)
}

func TestDeleteScaler_Success(t *testing.T) {
	fake := &FakeScalerPort{}
	svc := newTestService(&FakeServerPort{}, &FakeFleetPort{}, &FakeGamePort{}, fake, &FakePodPort{})

	resp, err := svc.DeleteScaler(context.Background(), gen.DeleteScalerRequestObject{
		Body: &gen.DeleteObjectRequest{Name: "test-scaler", Namespace: "default"},
	})

	require.NoError(t, err)
	assert.IsType(t, gen.DeleteScaler204Response{}, resp)
	assert.True(t, fake.DeleteScalerCalled)
}

func TestDeleteScaler_KubeError(t *testing.T) {
	fake := &FakeScalerPort{DeleteScalerErr: errors.New("not found")}
	svc := newTestService(&FakeServerPort{}, &FakeFleetPort{}, &FakeGamePort{}, fake, &FakePodPort{})

	resp, err := svc.DeleteScaler(context.Background(), gen.DeleteScalerRequestObject{
		Body: &gen.DeleteObjectRequest{Name: "test-scaler", Namespace: "default"},
	})

	require.NoError(t, err)
	assert.IsType(t, gen.DeleteScaler500JSONResponse{}, resp)
}
