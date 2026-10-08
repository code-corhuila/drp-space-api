package app

import (
	"context"
	"testing"
	"time"

	"github.com/code-corhuila/drp-space-api/internal/domain"
)

type stubSpaces []domain.Space

func (s stubSpaces) List(context.Context) ([]domain.Space, error) {
	return append([]domain.Space{}, s...), nil
}

func (s stubSpaces) FindByID(_ context.Context, id string) (domain.Space, error) {
	for _, sp := range s {
		if sp.ID == id {
			return sp, nil
		}
	}
	return domain.Space{}, ErrNotFound
}

func TestCatalogListOrderAndEmptyPage(t *testing.T) {
	now := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	norte, _ := domain.NewSpace("11111111-1111-1111-1111-111111111111", "Sala Norte", "MEETING_ROOM", "", 8, now)
	aula, _ := domain.NewSpace("22222222-2222-2222-2222-222222222222", "Aula Magna", "AUDITORIUM", "", 80, now)
	cat := Catalog{Spaces: stubSpaces{norte, aula}}
	page, err := cat.List(context.Background(), ListFilter{Page: 1, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 || page.Items[0].Name != "Aula Magna" || page.Items[1].Name != "Sala Norte" {
		t.Fatalf("order %+v", page)
	}
	empty, err := cat.List(context.Background(), ListFilter{Page: 9, Limit: 20})
	if err != nil || len(empty.Items) != 0 || empty.Total != 2 || empty.TotalPages != 1 {
		t.Fatalf("empty page %+v %v", empty, err)
	}
}

func TestCatalogRejectsBadPage(t *testing.T) {
	cat := Catalog{Spaces: stubSpaces{}}
	if _, err := cat.List(context.Background(), ListFilter{Page: 0, Limit: 20}); err != ErrValidation {
		t.Fatalf("page %v", err)
	}
	if _, err := cat.List(context.Background(), ListFilter{Page: 1, Limit: 101}); err != ErrValidation {
		t.Fatalf("limit %v", err)
	}
}
