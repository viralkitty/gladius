package build

import "testing"

func TestNewBuild(t *testing.T) {
	sha := "xyz"
	task := newTaskOrFatal(t, title)
	if task.Title != title {
		t.Errorf("expected title %q, got %q", title, task.Title)
	}
	if task.Done {
		t.Errorf("new task is done")
	}
}