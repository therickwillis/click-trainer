package events

type SceneChangeEvent struct {
	Scene string
}

var SceneChanges = make(chan SceneChangeEvent, 10)
