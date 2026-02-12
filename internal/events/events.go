package events

type SceneChangeEvent struct {
	Scene string
}

type Bus struct {
	SceneChanges chan SceneChangeEvent
}

func NewBus() *Bus {
	return &Bus{
		SceneChanges: make(chan SceneChangeEvent, 10),
	}
}
