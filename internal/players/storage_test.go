package players

import (
	"sync"
	"testing"
)

func TestNewStore(t *testing.T) {
	s := NewStore()
	if s == nil {
		t.Fatal("NewStore() returned nil")
	}
	list := s.GetList()
	if len(list) != 0 {
		t.Errorf("new store should be empty, got %d players", len(list))
	}
}

func TestStore_Add(t *testing.T) {
	s := NewStore()
	p := s.Add("id1", "Alice")

	if p.ID != "id1" {
		t.Errorf("player ID = %q, want %q", p.ID, "id1")
	}
	if p.Name != "Alice" {
		t.Errorf("player Name = %q, want %q", p.Name, "Alice")
	}
	if p.Color == "" {
		t.Error("player Color should not be empty")
	}
	if p.Score != 0 {
		t.Errorf("player Score = %d, want 0", p.Score)
	}
	if p.Ready {
		t.Error("player Ready should be false")
	}
}

func TestStore_Get(t *testing.T) {
	s := NewStore()
	s.Add("id1", "Alice")

	p := s.Get("id1")
	if p == nil {
		t.Fatal("Get returned nil for existing player")
	}
	if p.Name != "Alice" {
		t.Errorf("Name = %q, want %q", p.Name, "Alice")
	}

	p2 := s.Get("nonexistent")
	if p2 != nil {
		t.Error("Get should return nil for nonexistent player")
	}
}

func TestStore_GetList(t *testing.T) {
	s := NewStore()
	s.Add("id1", "Alice")
	s.Add("id2", "Bob")

	list := s.GetList()
	if len(list) != 2 {
		t.Errorf("GetList() returned %d players, want 2", len(list))
	}
}

func TestStore_UpdateScore(t *testing.T) {
	s := NewStore()
	s.Add("id1", "Alice")

	p := s.UpdateScore("id1", 10)
	if p.Score != 10 {
		t.Errorf("Score = %d, want 10", p.Score)
	}

	p = s.UpdateScore("id1", 5)
	if p.Score != 15 {
		t.Errorf("Score = %d, want 15", p.Score)
	}

	p = s.UpdateScore("nonexistent", 5)
	if p != nil {
		t.Error("UpdateScore should return nil for nonexistent player")
	}
}

func TestStore_SetReady(t *testing.T) {
	s := NewStore()
	s.Add("id1", "Alice")

	p := s.SetReady("id1", true)
	if !p.Ready {
		t.Error("player should be ready")
	}

	p = s.SetReady("id1", false)
	if p.Ready {
		t.Error("player should not be ready")
	}

	p = s.SetReady("nonexistent", true)
	if p != nil {
		t.Error("SetReady should return nil for nonexistent player")
	}
}

func TestStore_AllReady(t *testing.T) {
	s := NewStore()

	// Empty store
	if s.AllReady() {
		t.Error("AllReady should be false for empty store")
	}

	s.Add("id1", "Alice")
	s.Add("id2", "Bob")

	// No one ready
	if s.AllReady() {
		t.Error("AllReady should be false when no one is ready")
	}

	// One ready
	s.SetReady("id1", true)
	if s.AllReady() {
		t.Error("AllReady should be false when only one player is ready")
	}

	// All ready
	s.SetReady("id2", true)
	if !s.AllReady() {
		t.Error("AllReady should be true when all players are ready")
	}
}

func TestStore_ValidateSession(t *testing.T) {
	s := NewStore()
	s.Add("id1", "Alice")

	if !s.ValidateSession("id1") {
		t.Error("ValidateSession should return true for existing player")
	}
	if s.ValidateSession("nonexistent") {
		t.Error("ValidateSession should return false for nonexistent player")
	}
}

func TestStore_Remove(t *testing.T) {
	s := NewStore()
	s.Add("id1", "Alice")
	s.Add("id2", "Bob")

	if !s.Remove("id1") {
		t.Error("Remove should return true for existing player")
	}
	if s.Get("id1") != nil {
		t.Error("player should be nil after removal")
	}
	if len(s.GetList()) != 1 {
		t.Errorf("expected 1 player after removal, got %d", len(s.GetList()))
	}

	if s.Remove("nonexistent") {
		t.Error("Remove should return false for nonexistent player")
	}
}

func TestStore_Count(t *testing.T) {
	s := NewStore()
	if s.Count() != 0 {
		t.Errorf("Count = %d, want 0", s.Count())
	}

	s.Add("id1", "Alice")
	if s.Count() != 1 {
		t.Errorf("Count = %d, want 1", s.Count())
	}

	s.Add("id2", "Bob")
	if s.Count() != 2 {
		t.Errorf("Count = %d, want 2", s.Count())
	}

	s.Remove("id1")
	if s.Count() != 1 {
		t.Errorf("Count = %d, want 1 after removal", s.Count())
	}
}

func TestStore_ResetAll(t *testing.T) {
	s := NewStore()
	s.Add("id1", "Alice")
	s.Add("id2", "Bob")
	s.UpdateScore("id1", 100)
	s.SetReady("id1", true)
	s.SetReady("id2", true)

	s.ResetAll()

	p1 := s.Get("id1")
	p2 := s.Get("id2")

	if p1.Score != 0 || p2.Score != 0 {
		t.Error("scores should be reset to 0")
	}
	if p1.Ready || p2.Ready {
		t.Error("ready states should be reset to false")
	}
	// Players should still exist
	if len(s.GetList()) != 2 {
		t.Error("players should still exist after reset")
	}
}

func TestStore_ConcurrentAccess(t *testing.T) {
	s := NewStore()
	s.Add("id1", "Alice")

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.UpdateScore("id1", 1)
		}()
	}
	wg.Wait()

	p := s.Get("id1")
	if p.Score != 100 {
		t.Errorf("concurrent Score = %d, want 100", p.Score)
	}
}
