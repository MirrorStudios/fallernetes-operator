package service

import (
	"context"

	"github.com/MirrorStudios/fallernetes-operator/service/internal/gen"
)

func (s *OperatorService) CreateFleet(ctx context.Context, req gen.CreateFleetRequestObject) (gen.CreateFleetResponseObject, error) {
	labels := map[string]string{}
	if req.Body.Labels != nil {
		labels = *req.Body.Labels
	}
	if err := s.fleet.CreateFleet(ctx, req.Body.Name, req.Body.Namespace, labels, req.Body.Spec); err != nil {
		return gen.CreateFleet500JSONResponse{InternalErrorJSONResponse: gen.InternalErrorJSONResponse{Message: "error creating fleet", Error: strPtr(err.Error())}}, nil
	}
	return gen.CreateFleet201JSONResponse(*req.Body), nil
}

func (s *OperatorService) DeleteFleet(ctx context.Context, req gen.DeleteFleetRequestObject) (gen.DeleteFleetResponseObject, error) {
	force := false
	if req.Body.Force != nil {
		force = *req.Body.Force
	}
	if err := s.fleet.DeleteFleet(ctx, req.Body.Name, req.Body.Namespace, force); err != nil {
		return gen.DeleteFleet500JSONResponse{InternalErrorJSONResponse: gen.InternalErrorJSONResponse{Message: "error deleting fleet", Error: strPtr(err.Error())}}, nil
	}
	return gen.DeleteFleet204Response{}, nil
}

func (s *OperatorService) PatchFleetReplicas(ctx context.Context, req gen.PatchFleetReplicasRequestObject) (gen.PatchFleetReplicasResponseObject, error) {
	if err := s.fleet.PatchFleetReplicas(ctx, req.Body.Name, req.Body.Namespace, req.Body.Replicas, req.Body.MinReplicas, req.Body.MaxReplicas); err != nil {
		return gen.PatchFleetReplicas500JSONResponse{InternalErrorJSONResponse: gen.InternalErrorJSONResponse{Message: "error patching fleet replicas", Error: strPtr(err.Error())}}, nil
	}
	return gen.PatchFleetReplicas200Response{}, nil
}
