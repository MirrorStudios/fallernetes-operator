package test

import (
	"context"
	"testing"

	"github.com/MirrorStudios/fallernetes-sidecar/internal/gen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAllowDelete_ReturnsFalse(t *testing.T) {
	state := &FakeStatePort{deleteAllowed: false}
	svc := newTestService(state)
	resp, err := svc.GetAllowDelete(context.Background(), gen.GetAllowDeleteRequestObject{})
	require.NoError(t, err)
	assert.Equal(t, gen.GetAllowDelete200JSONResponse{Allowed: false}, resp)
}

func TestGetAllowDelete_ReturnsTrue(t *testing.T) {
	state := &FakeStatePort{deleteAllowed: true}
	svc := newTestService(state)
	resp, err := svc.GetAllowDelete(context.Background(), gen.GetAllowDeleteRequestObject{})
	require.NoError(t, err)
	assert.Equal(t, gen.GetAllowDelete200JSONResponse{Allowed: true}, resp)
}

func TestSetAllowDelete_SetsTrue(t *testing.T) {
	state := &FakeStatePort{deleteAllowed: false}
	svc := newTestService(state)
	body := gen.AllowDeleteRequest{Allowed: true}
	resp, err := svc.SetAllowDelete(context.Background(), gen.SetAllowDeleteRequestObject{Body: &body})
	require.NoError(t, err)
	assert.IsType(t, gen.SetAllowDelete200Response{}, resp)
	assert.True(t, state.IsDeleteAllowed())
}

func TestSetAllowDelete_SetsFalse(t *testing.T) {
	state := &FakeStatePort{deleteAllowed: true}
	svc := newTestService(state)
	body := gen.AllowDeleteRequest{Allowed: false}
	resp, err := svc.SetAllowDelete(context.Background(), gen.SetAllowDeleteRequestObject{Body: &body})
	require.NoError(t, err)
	assert.IsType(t, gen.SetAllowDelete200Response{}, resp)
	assert.False(t, state.IsDeleteAllowed())
}
