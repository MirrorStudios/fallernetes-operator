package test

import (
	"context"
	"testing"

	"github.com/MirrorStudios/fallernetes-sidecar/internal/gen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetShutdown_ReturnsFalse(t *testing.T) {
	state := &FakeStatePort{shutdownRequested: false}
	svc := newTestService(state)
	resp, err := svc.GetShutdown(context.Background(), gen.GetShutdownRequestObject{})
	require.NoError(t, err)
	assert.Equal(t, gen.GetShutdown200JSONResponse{Shutdown: false}, resp)
}

func TestGetShutdown_ReturnsTrue(t *testing.T) {
	state := &FakeStatePort{shutdownRequested: true}
	svc := newTestService(state)
	resp, err := svc.GetShutdown(context.Background(), gen.GetShutdownRequestObject{})
	require.NoError(t, err)
	assert.Equal(t, gen.GetShutdown200JSONResponse{Shutdown: true}, resp)
}

func TestSetShutdown_SetsTrue(t *testing.T) {
	state := &FakeStatePort{shutdownRequested: false}
	svc := newTestService(state)
	body := gen.ShutdownRequest{Shutdown: true}
	resp, err := svc.SetShutdown(context.Background(), gen.SetShutdownRequestObject{Body: &body})
	require.NoError(t, err)
	assert.IsType(t, gen.SetShutdown200Response{}, resp)
	assert.True(t, state.IsShutdownRequested())
}

func TestSetShutdown_SetsFalse(t *testing.T) {
	state := &FakeStatePort{shutdownRequested: true}
	svc := newTestService(state)
	body := gen.ShutdownRequest{Shutdown: false}
	resp, err := svc.SetShutdown(context.Background(), gen.SetShutdownRequestObject{Body: &body})
	require.NoError(t, err)
	assert.IsType(t, gen.SetShutdown200Response{}, resp)
	assert.False(t, state.IsShutdownRequested())
}
