package app

import (
	"context"

	"github.com/code-corhuila/drp-space-api/internal/domain"
)

type SpaceRepository interface {
	List(ctx context.Context) ([]domain.Space, error)
	FindByID(ctx context.Context, id string) (domain.Space, error)
}

type TokenVerifier interface {
	Parse(ctx context.Context, raw string) (subject string, err error)
}
