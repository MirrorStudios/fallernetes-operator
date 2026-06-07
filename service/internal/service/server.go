package service

import (
	"context"

	"github.com/MirrorStudios/fallernetes-service/internal/gen"
)

func (s *OperatorService) CreateServer(ctx context.Context, req gen.CreateServerRequestObject) (gen.CreateServerResponseObject, error) {
	labels := map[string]string{}
	if req.Body.Labels != nil {
		labels = *req.Body.Labels
	}
	if err := s.server.CreateServer(ctx, req.Body.Name, req.Body.Namespace, labels, req.Body.Spec); err != nil {
		return gen.CreateServer500JSONResponse{InternalErrorJSONResponse: gen.InternalErrorJSONResponse{Message: "error creating server", Error: strPtr(err.Error())}}, nil
	}
	return gen.CreateServer201JSONResponse(*req.Body), nil
}

func (s *OperatorService) DeleteServer(ctx context.Context, req gen.DeleteServerRequestObject) (gen.DeleteServerResponseObject, error) {
	force := false
	if req.Body.Force != nil {
		force = *req.Body.Force
	}
	if err := s.server.DeleteServer(ctx, req.Body.Name, req.Body.Namespace, force); err != nil {
		return gen.DeleteServer500JSONResponse{InternalErrorJSONResponse: gen.InternalErrorJSONResponse{Message: "error deleting server", Error: strPtr(err.Error())}}, nil
	}
	return gen.DeleteServer204Response{}, nil
}
