package app

import (
	"context"
	"sort"

	"github.com/code-corhuila/drp-space-api/internal/domain"
)

type ListFilter struct {
	Kind      *domain.Kind
	Available *bool
	Page      int
	Limit     int
}

type SpacePage struct {
	Items      []domain.Space
	Page       int
	Limit      int
	Total      int
	TotalPages int
}

type Catalog struct {
	Spaces SpaceRepository
}

func (c Catalog) List(ctx context.Context, f ListFilter) (SpacePage, error) {
	if f.Page < 1 || f.Limit < 1 || f.Limit > 100 {
		return SpacePage{}, ErrValidation
	}
	all, err := c.Spaces.List(ctx)
	if err != nil {
		return SpacePage{}, err
	}
	filtered := make([]domain.Space, 0, len(all))
	for _, s := range all {
		if s.DeletedAt != nil {
			continue
		}
		if f.Kind != nil && s.Kind != *f.Kind {
			continue
		}
		if f.Available != nil && s.Active != *f.Available {
			continue
		}
		filtered = append(filtered, s)
	}
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].Name != filtered[j].Name {
			return filtered[i].Name < filtered[j].Name
		}
		return filtered[i].ID < filtered[j].ID
	})
	total := len(filtered)
	totalPages := 0
	if total > 0 {
		totalPages = (total + f.Limit - 1) / f.Limit
	}
	start := (f.Page - 1) * f.Limit
	if start >= total {
		return SpacePage{Items: []domain.Space{}, Page: f.Page, Limit: f.Limit, Total: total, TotalPages: totalPages}, nil
	}
	end := start + f.Limit
	if end > total {
		end = total
	}
	return SpacePage{Items: filtered[start:end], Page: f.Page, Limit: f.Limit, Total: total, TotalPages: totalPages}, nil
}

func (c Catalog) Get(ctx context.Context, id string) (domain.Space, error) {
	s, err := c.Spaces.FindByID(ctx, id)
	if err != nil {
		return domain.Space{}, err
	}
	if s.DeletedAt != nil {
		return domain.Space{}, ErrNotFound
	}
	return s, nil
}
