package service

import (
	"context"

	"github.com/MirrorStudios/fallernetes-operator/service/internal/gen"
)

func (s *OperatorService) CreateScaler(ctx context.Context, req gen.CreateScalerRequestObject) (gen.CreateScalerResponseObject, error) {
	labels := map[string]string{}
	if req.Body.Labels != nil {
		labels = *req.Body.Labels
	}
	if err := s.scaler.CreateScaler(ctx, req.Body.Name, req.Body.Namespace, labels, req.Body.Spec); err != nil {
		return gen.CreateScaler500JSONResponse{InternalErrorJSONResponse: gen.InternalErrorJSONResponse{Message: "error creating scaler", Error: strPtr(err.Error())}}, nil
	}
	return gen.CreateScaler201JSONResponse(*req.Body), nil
}

func (s *OperatorService) DeleteScaler(ctx context.Context, req gen.DeleteScalerRequestObject) (gen.DeleteScalerResponseObject, error) {
	if err := s.scaler.DeleteScaler(ctx, req.Body.Name, req.Body.Namespace); err != nil {
		return gen.DeleteScaler500JSONResponse{InternalErrorJSONResponse: gen.InternalErrorJSONResponse{Message: "error deleting scaler", Error: strPtr(err.Error())}}, nil
	}
	return gen.DeleteScaler204Response{}, nil
}
