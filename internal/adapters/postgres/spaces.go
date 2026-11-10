package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/code-corhuila/drp-space-api/internal/app"
	"github.com/code-corhuila/drp-space-api/internal/domain"
)

type Spaces struct {
	DB *sql.DB
}

func NewSpaces(db *sql.DB) *Spaces {
	return &Spaces{DB: db}
}

func (s *Spaces) List(ctx context.Context) ([]domain.Space, error) {
	rows, err := s.DB.QueryContext(ctx, `
SELECT id, name, kind, COALESCE(description, ''), capacity, active, created_at, updated_at, deleted_at
FROM spaces WHERE deleted_at IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Space{}
	for rows.Next() {
		got, err := scanSpace(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, got)
	}
	return out, rows.Err()
}

func (s *Spaces) FindByID(ctx context.Context, id string) (domain.Space, error) {
	row := s.DB.QueryRowContext(ctx, `
SELECT id, name, kind, COALESCE(description, ''), capacity, active, created_at, updated_at, deleted_at
FROM spaces WHERE id = $1 AND deleted_at IS NULL`, id)
	got, err := scanSpace(row)
	if err == sql.ErrNoRows {
		return domain.Space{}, app.ErrNotFound
	}
	return got, err
}

func (s *Spaces) Save(ctx context.Context, sp domain.Space) error {
	_, err := s.DB.ExecContext(ctx, `
INSERT INTO spaces (id, name, kind, description, capacity, active, created_at, updated_at, deleted_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULL)`,
		sp.ID, sp.Name, string(sp.Kind), sp.Description, sp.Capacity, sp.Active, sp.CreatedAt, sp.UpdatedAt)
	return err
}

func (s *Spaces) ListBlocks(ctx context.Context, spaceID string) ([]domain.BlockedPeriod, error) {
	rows, err := s.DB.QueryContext(ctx, `
SELECT id, space_id, start_at, end_at, COALESCE(reason, ''), created_at
FROM blocked_periods WHERE space_id = $1`, spaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.BlockedPeriod{}
	for rows.Next() {
		var b domain.BlockedPeriod
		var start, end time.Time
		if err := rows.Scan(&b.ID, &b.SpaceID, &start, &end, &b.Reason, &b.CreatedAt); err != nil {
			return nil, err
		}
		p, err := domain.NewDateTimeRange(start.UTC(), end.UTC())
		if err != nil {
			return nil, err
		}
		b.Period = p
		b.CreatedAt = b.CreatedAt.UTC()
		out = append(out, b)
	}
	return out, rows.Err()
}

func (s *Spaces) SaveBlock(ctx context.Context, b domain.BlockedPeriod) error {
	_, err := s.DB.ExecContext(ctx, `
INSERT INTO blocked_periods (id, space_id, start_at, end_at, reason, created_at)
VALUES ($1, $2, $3, $4, $5, $6)`,
		b.ID, b.SpaceID, b.Period.Start, b.Period.End, b.Reason, b.CreatedAt)
	return err
}

type scanner interface {
	Scan(dest ...any) error
}

func scanSpace(sc scanner) (domain.Space, error) {
	var got domain.Space
	var kind string
	var deleted sql.NullTime
	if err := sc.Scan(&got.ID, &got.Name, &kind, &got.Description, &got.Capacity, &got.Active, &got.CreatedAt, &got.UpdatedAt, &deleted); err != nil {
		return domain.Space{}, err
	}
	k, err := domain.ParseKind(kind)
	if err != nil {
		return domain.Space{}, err
	}
	got.Kind = k
	got.CreatedAt = got.CreatedAt.UTC()
	got.UpdatedAt = got.UpdatedAt.UTC()
	if deleted.Valid {
		t := deleted.Time.UTC()
		got.DeletedAt = &t
	}
	return got, nil
}
