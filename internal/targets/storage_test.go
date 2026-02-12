package targets

import (
	"testing"
)

func TestNewStore(t *testing.T) {
	s := NewStore()
	if s == nil {
		t.Fatal("NewStore() returned nil")
	}
	list := s.GetList()
	if len(list) != 0 {
		t.Errorf("new store should be empty, got %d targets", len(list))
	}
}

func TestStore_Add(t *testing.T) {
	s := NewStore()
	target := s.Add()

	if target.ID != 1 {
		t.Errorf("first target ID = %d, want 1", target.ID)
	}
	if target.X < 0 || target.X >= GameWidth {
		t.Errorf("target X = %d, out of bounds", target.X)
	}
	if target.Y < 0 || target.Y >= GameHeight {
		t.Errorf("target Y = %d, out of bounds", target.Y)
	}
	if target.Size < MinTargetSize || target.Size > MaxTargetSize {
		t.Errorf("target Size = %d, out of bounds [%d, %d]", target.Size, MinTargetSize, MaxTargetSize)
	}
	if target.Color == "" {
		t.Error("target Color should not be empty")
	}
	if target.Dead {
		t.Error("new target should not be dead")
	}
}

func TestStore_Add_AutoIncrement(t *testing.T) {
	s := NewStore()
	t1 := s.Add()
	t2 := s.Add()
	t3 := s.Add()

	if t1.ID != 1 || t2.ID != 2 || t3.ID != 3 {
		t.Errorf("IDs = %d, %d, %d; want 1, 2, 3", t1.ID, t2.ID, t3.ID)
	}
}

func TestStore_Kill(t *testing.T) {
	s := NewStore()
	target := s.Add()

	s.Kill(target.ID)

	list := s.GetList()
	if len(list) != 0 {
		t.Errorf("killed target should not appear in GetList, got %d", len(list))
	}
}

func TestStore_Kill_Nonexistent(t *testing.T) {
	s := NewStore()
	// Should not panic
	s.Kill(999)
}

func TestStore_GetList(t *testing.T) {
	s := NewStore()
	s.Add()
	s.Add()
	s.Add()

	list := s.GetList()
	if len(list) != 3 {
		t.Errorf("GetList() returned %d targets, want 3", len(list))
	}

	// Kill one
	s.Kill(2)
	list = s.GetList()
	if len(list) != 2 {
		t.Errorf("GetList() after kill returned %d targets, want 2", len(list))
	}
}

func TestStore_Clear(t *testing.T) {
	s := NewStore()
	s.Add()
	s.Add()
	s.Add()

	s.Clear()

	list := s.GetList()
	if len(list) != 0 {
		t.Errorf("after Clear(), got %d targets, want 0", len(list))
	}

	// IDs should reset
	newTarget := s.Add()
	if newTarget.ID != 1 {
		t.Errorf("after Clear(), new ID = %d, want 1", newTarget.ID)
	}
}
