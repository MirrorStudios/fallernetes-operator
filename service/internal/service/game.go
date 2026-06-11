package service

import (
	"context"

	"github.com/MirrorStudios/fallernetes-operator/service/internal/gen"
)

func (s *OperatorService) CreateGame(ctx context.Context, req gen.CreateGameRequestObject) (gen.CreateGameResponseObject, error) {
	labels := map[string]string{}
	if req.Body.Labels != nil {
		labels = *req.Body.Labels
	}
	if err := s.game.CreateGame(ctx, req.Body.Name, req.Body.Namespace, labels, req.Body.Spec); err != nil {
		return gen.CreateGame500JSONResponse{InternalErrorJSONResponse: gen.InternalErrorJSONResponse{Message: "error creating game", Error: strPtr(err.Error())}}, nil
	}
	return gen.CreateGame201JSONResponse(*req.Body), nil
}

func (s *OperatorService) DeleteGame(ctx context.Context, req gen.DeleteGameRequestObject) (gen.DeleteGameResponseObject, error) {
	force := false
	if req.Body.Force != nil {
		force = *req.Body.Force
	}
	if err := s.game.DeleteGame(ctx, req.Body.Name, req.Body.Namespace, force); err != nil {
		return gen.DeleteGame500JSONResponse{InternalErrorJSONResponse: gen.InternalErrorJSONResponse{Message: "error deleting game", Error: strPtr(err.Error())}}, nil
	}
	return gen.DeleteGame204Response{}, nil
}
