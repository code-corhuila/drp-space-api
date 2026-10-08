package domain

import (
	"testing"
	"time"
)

func TestNewSpaceAndBlock(t *testing.T) {
	now := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	s, err := NewSpace("11111111-1111-1111-1111-111111111111", "Sala Norte", "MEETING_ROOM", "8p", 8, now)
	if err != nil {
		t.Fatal(err)
	}
	start := now.Add(time.Hour)
	end := start.Add(2 * time.Hour)
	b, err := NewBlockedPeriod("b1", s.ID, "maintenance", start, end, now)
	if err != nil {
		t.Fatal(err)
	}
	p, _ := NewDateTimeRange(start.Add(30*time.Minute), end.Add(30*time.Minute))
	if !s.PeriodBlocked([]BlockedPeriod{b}, p) {
		t.Fatal("expected blocked overlap")
	}
	free, _ := NewDateTimeRange(end, end.Add(time.Hour))
	if s.PeriodBlocked([]BlockedPeriod{b}, free) {
		t.Fatal("expected free after block")
	}
	inactive := s.Deactivate(now)
	if !inactive.PeriodBlocked(nil, free) {
		t.Fatal("inactive must not be offered")
	}
}

func TestSpaceRejects(t *testing.T) {
	now := time.Now().UTC()
	if _, err := NewSpace("id", "Sala", "HALL", "", 8, now); err == nil {
		t.Fatal("bad kind")
	}
	if _, err := NewSpace("id", "Sala", "WORKSTATION", "", 0, now); err == nil {
		t.Fatal("capacity")
	}
	start := now
	if _, err := NewBlockedPeriod("b", "s", "", start, start, now); err == nil {
		t.Fatal("end must be after start")
	}
}
