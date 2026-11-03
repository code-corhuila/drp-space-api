package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sort"
	"strings"
	"time"

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
	Now    func() time.Time
	NewID  func() string
}

type BlockPage struct {
	Items      []domain.BlockedPeriod
	Page       int
	Limit      int
	Total      int
	TotalPages int
}

func (c Catalog) now() time.Time {
	if c.Now == nil {
		return time.Now().UTC()
	}
	return c.Now().UTC()
}

func (c Catalog) newID() string {
	if c.NewID != nil {
		return c.NewID()
	}
	var buf [16]byte
	_, _ = rand.Read(buf[:])
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80
	h := hex.EncodeToString(buf[:])
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
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

type CreateSpaceCommand struct {
	Actor       Principal
	Name        string
	Kind        string
	Description string
	Capacity    int
}

func (c Catalog) Create(ctx context.Context, cmd CreateSpaceCommand) (domain.Space, error) {
	if !cmd.Actor.Admin() {
		return domain.Space{}, ErrForbidden
	}
	s, err := domain.NewSpace(c.newID(), cmd.Name, cmd.Kind, strings.TrimSpace(cmd.Description), cmd.Capacity, c.now())
	if err != nil {
		return domain.Space{}, ErrValidation
	}
	all, err := c.Spaces.List(ctx)
	if err != nil {
		return domain.Space{}, err
	}
	for _, existing := range all {
		if existing.DeletedAt != nil {
			continue
		}
		if strings.EqualFold(existing.Name, s.Name) {
			return domain.Space{}, ErrConflict
		}
	}
	if err := c.Spaces.Save(ctx, s); err != nil {
		return domain.Space{}, err
	}
	return s, nil
}

func (c Catalog) ListBlocks(ctx context.Context, actor Principal, spaceID string, page, limit int) (BlockPage, error) {
	if !actor.Admin() {
		return BlockPage{}, ErrForbidden
	}
	if page < 1 || limit < 1 || limit > 100 {
		return BlockPage{}, ErrValidation
	}
	if _, err := c.Get(ctx, spaceID); err != nil {
		return BlockPage{}, err
	}
	all, err := c.Spaces.ListBlocks(ctx, spaceID)
	if err != nil {
		return BlockPage{}, err
	}
	sort.Slice(all, func(i, j int) bool {
		if !all[i].Period.Start.Equal(all[j].Period.Start) {
			return all[i].Period.Start.Before(all[j].Period.Start)
		}
		return all[i].ID < all[j].ID
	})
	total := len(all)
	totalPages := 0
	if total > 0 {
		totalPages = (total + limit - 1) / limit
	}
	start := (page - 1) * limit
	if start >= total {
		return BlockPage{Items: []domain.BlockedPeriod{}, Page: page, Limit: limit, Total: total, TotalPages: totalPages}, nil
	}
	end := start + limit
	if end > total {
		end = total
	}
	return BlockPage{Items: all[start:end], Page: page, Limit: limit, Total: total, TotalPages: totalPages}, nil
}

type BlockPeriodCommand struct {
	Actor   Principal
	SpaceID string
	StartAt time.Time
	EndAt   time.Time
	Reason  string
}

func (c Catalog) BlockPeriod(ctx context.Context, cmd BlockPeriodCommand) (domain.BlockedPeriod, error) {
	if !cmd.Actor.Admin() {
		return domain.BlockedPeriod{}, ErrForbidden
	}
	if _, err := c.Get(ctx, cmd.SpaceID); err != nil {
		return domain.BlockedPeriod{}, err
	}
	b, err := domain.NewBlockedPeriod(c.newID(), cmd.SpaceID, strings.TrimSpace(cmd.Reason), cmd.StartAt, cmd.EndAt, c.now())
	if err != nil {
		return domain.BlockedPeriod{}, ErrValidation
	}
	if err := c.Spaces.SaveBlock(ctx, b); err != nil {
		return domain.BlockedPeriod{}, err
	}
	return b, nil
}
