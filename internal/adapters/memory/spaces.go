package memory

import (
	"context"
	"sync"
	"time"

	"github.com/code-corhuila/drp-space-api/internal/app"
	"github.com/code-corhuila/drp-space-api/internal/domain"
)

// Spaces is an in-memory catalog. Flyway persistence is a later feat.
type Spaces struct {
	mu   sync.RWMutex
	byID map[string]domain.Space
}

func Corte2() (*Spaces, error) {
	now := time.Unix(0, 0).UTC()
	norte, err := domain.NewSpace("11111111-1111-1111-1111-111111111111", "Sala Norte", "MEETING_ROOM", "8 personas", 8, now)
	if err != nil {
		return nil, err
	}
	aula, err := domain.NewSpace("22222222-2222-2222-2222-222222222222", "Aula Magna", "AUDITORIUM", "", 80, now)
	if err != nil {
		return nil, err
	}
	cub, err := domain.NewSpace("33333333-3333-3333-3333-333333333333", "Cubículo 1", "WORKSTATION", "", 1, now)
	if err != nil {
		return nil, err
	}
	office, err := domain.NewSpace("55555555-5555-5555-5555-555555555555", "Oficina Cerrada", "PRIVATE_OFFICE", "", 4, now)
	if err != nil {
		return nil, err
	}
	office = office.Deactivate(now)
	r := &Spaces{byID: map[string]domain.Space{}}
	for _, s := range []domain.Space{norte, aula, cub, office} {
		r.byID[s.ID] = s
	}
	return r, nil
}

func (r *Spaces) List(_ context.Context) ([]domain.Space, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.Space, 0, len(r.byID))
	for _, s := range r.byID {
		out = append(out, s)
	}
	return out, nil
}

func (r *Spaces) FindByID(_ context.Context, id string) (domain.Space, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.byID[id]
	if !ok {
		return domain.Space{}, app.ErrNotFound
	}
	return s, nil
}
