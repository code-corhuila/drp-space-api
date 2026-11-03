package app

import (
	"context"

	"github.com/code-corhuila/drp-space-api/internal/domain"
)

type Principal struct {
	Subject string
	Role    string
}

func (p Principal) Admin() bool { return p.Role == "ADMIN" }

type SpaceRepository interface {
	List(ctx context.Context) ([]domain.Space, error)
	FindByID(ctx context.Context, id string) (domain.Space, error)
	Save(ctx context.Context, s domain.Space) error
	ListBlocks(ctx context.Context, spaceID string) ([]domain.BlockedPeriod, error)
	SaveBlock(ctx context.Context, b domain.BlockedPeriod) error
}

type TokenVerifier interface {
	Parse(ctx context.Context, raw string) (Principal, error)
}
