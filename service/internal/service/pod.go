package service

import (
	"context"

	"github.com/MirrorStudios/fallernetes-service/internal/gen"
)

func (s *OperatorService) AddPodLabel(ctx context.Context, req gen.AddPodLabelRequestObject) (gen.AddPodLabelResponseObject, error) {
	if err := s.pod.AddPodLabel(ctx, req.Body.ServerName, req.Body.Namespace, req.Body.Key, req.Body.Value); err != nil {
		return gen.AddPodLabel500JSONResponse{InternalErrorJSONResponse: gen.InternalErrorJSONResponse{Message: "error adding pod label", Error: strPtr(err.Error())}}, nil
	}
	return gen.AddPodLabel200Response{}, nil
}

func (s *OperatorService) RemovePodLabel(ctx context.Context, req gen.RemovePodLabelRequestObject) (gen.RemovePodLabelResponseObject, error) {
	if err := s.pod.RemovePodLabel(ctx, req.Body.ServerName, req.Body.Namespace, req.Body.Key); err != nil {
		return gen.RemovePodLabel500JSONResponse{InternalErrorJSONResponse: gen.InternalErrorJSONResponse{Message: "error removing pod label", Error: strPtr(err.Error())}}, nil
	}
	return gen.RemovePodLabel204Response{}, nil
}
