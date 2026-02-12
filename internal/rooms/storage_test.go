package rooms

import (
	"clicktrainer/internal/gamedata"
	"sync"
	"testing"
)

func testConfig() gamedata.Config {
	return gamedata.Config{
		RoundDuration:  5,
		InitialTargets: 2,
		CountdownSecs:  1,
	}
}

func TestNewStore(t *testing.T) {
	s := NewStore(testConfig())
	if s == nil {
		t.Fatal("NewStore() returned nil")
	}
	if len(s.List()) != 0 {
		t.Error("new store should have no rooms")
	}
}

func TestStore_Create(t *testing.T) {
	s := NewStore(testConfig())
	room, err := s.Create("host-1")
	if err != nil {
		t.Fatal(err)
	}
	if room == nil {
		t.Fatal("Create() returned nil room")
	}
	if room.Code == "" {
		t.Error("room code should not be empty")
	}
	if room.HostID != "host-1" {
		t.Errorf("HostID = %q, want %q", room.HostID, "host-1")
	}
	if room.Game == nil {
		t.Error("room Game should not be nil")
	}
	if room.Broadcaster == nil {
		t.Error("room Broadcaster should not be nil")
	}
}

func TestStore_Get(t *testing.T) {
	s := NewStore(testConfig())
	room, _ := s.Create("host-1")

	got := s.Get(room.Code)
	if got == nil {
		t.Fatal("Get() returned nil for existing room")
	}
	if got.Code != room.Code {
		t.Errorf("Code = %q, want %q", got.Code, room.Code)
	}

	got = s.Get("ZZZZ")
	if got != nil {
		t.Error("Get() should return nil for nonexistent room")
	}
}

func TestStore_Delete(t *testing.T) {
	s := NewStore(testConfig())
	room, _ := s.Create("host-1")

	s.Delete(room.Code)

	if s.Get(room.Code) != nil {
		t.Error("room should be deleted")
	}
}

func TestStore_List(t *testing.T) {
	s := NewStore(testConfig())
	s.Create("host-1")
	s.Create("host-2")

	list := s.List()
	if len(list) != 2 {
		t.Errorf("List() returned %d rooms, want 2", len(list))
	}
}

func TestStore_ConcurrentAccess(t *testing.T) {
	s := NewStore(testConfig())

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Create("host")
		}()
	}
	wg.Wait()

	list := s.List()
	if len(list) != 50 {
		t.Errorf("concurrent creates: got %d rooms, want 50", len(list))
	}
}

func TestStore_RoomIsolation(t *testing.T) {
	s := NewStore(testConfig())
	room1, _ := s.Create("host-1")
	room2, _ := s.Create("host-2")

	room1.Game.Players.Add("p1", "Alice")
	room2.Game.Players.Add("p2", "Bob")

	// Players in room1 shouldn't be visible in room2
	r1Players := room1.Game.Players.GetList()
	r2Players := room2.Game.Players.GetList()

	if len(r1Players) != 1 || r1Players[0].Name != "Alice" {
		t.Error("room1 should only have Alice")
	}
	if len(r2Players) != 1 || r2Players[0].Name != "Bob" {
		t.Error("room2 should only have Bob")
	}
}
