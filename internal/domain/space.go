package domain

import (
	"strings"
	"time"
)

type DateTimeRange struct {
	Start time.Time
	End   time.Time
}

func NewDateTimeRange(start, end time.Time) (DateTimeRange, error) {
	if start.IsZero() || end.IsZero() || !end.After(start) {
		return DateTimeRange{}, ErrInvalidInput
	}
	return DateTimeRange{Start: start.UTC(), End: end.UTC()}, nil
}

func (r DateTimeRange) Overlaps(other DateTimeRange) bool {
	return r.Start.Before(other.End) && other.Start.Before(r.End)
}

type Kind string

const (
	KindWorkstation   Kind = "WORKSTATION"
	KindMeetingRoom   Kind = "MEETING_ROOM"
	KindPrivateOffice Kind = "PRIVATE_OFFICE"
	KindTrainingRoom  Kind = "TRAINING_ROOM"
	KindAuditorium    Kind = "AUDITORIUM"
)

func ParseKind(v string) (Kind, error) {
	switch Kind(v) {
	case KindWorkstation, KindMeetingRoom, KindPrivateOffice, KindTrainingRoom, KindAuditorium:
		return Kind(v), nil
	default:
		return "", ErrInvalidInput
	}
}

// Space is the catalog aggregate root. CONFIRMED overlap is not stored here.
type Space struct {
	ID          string
	Name        string
	Kind        Kind
	Description string
	Capacity    int
	Active      bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

func NewSpace(id, name, kind, description string, capacity int, now time.Time) (Space, error) {
	k, err := ParseKind(kind)
	if err != nil {
		return Space{}, err
	}
	n := strings.TrimSpace(name)
	if id == "" || n == "" || len(n) > 200 || capacity < 1 {
		return Space{}, ErrInvalidInput
	}
	return Space{
		ID:          id,
		Name:        n,
		Kind:        k,
		Description: description,
		Capacity:    capacity,
		Active:      true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (s Space) Deactivate(now time.Time) Space {
	s.Active = false
	s.UpdatedAt = now
	return s
}

type BlockedPeriod struct {
	ID        string
	SpaceID   string
	Period    DateTimeRange
	Reason    string
	CreatedAt time.Time
}

func NewBlockedPeriod(id, spaceID, reason string, start, end, now time.Time) (BlockedPeriod, error) {
	p, err := NewDateTimeRange(start, end)
	if err != nil {
		return BlockedPeriod{}, err
	}
	if id == "" || spaceID == "" || len(reason) > 500 {
		return BlockedPeriod{}, ErrInvalidInput
	}
	return BlockedPeriod{ID: id, SpaceID: spaceID, Period: p, Reason: reason, CreatedAt: now}, nil
}

func (s Space) PeriodBlocked(blocks []BlockedPeriod, period DateTimeRange) bool {
	if !s.Active || s.DeletedAt != nil {
		return true
	}
	for _, b := range blocks {
		if b.SpaceID == s.ID && b.Period.Overlaps(period) {
			return true
		}
	}
	return false
}
