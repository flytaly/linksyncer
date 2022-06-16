package imagesync

import (
	"reflect"
	"testing"
	"testing/fstest"
)

func TestProcessFiles(t *testing.T) {
	markdown := `
				![alt text](./assets/image01.png)
				![alt text](./assets/image02.png)
	`
	mapFS := fstest.MapFS{
		"notes/my/note.md":  {Data: []byte(markdown)},
		"notes/my/note2.md": {Data: []byte("![alt text](./assets/image02.png)")},
	}

	root := "/home/user/notes"

	iSync := New(mapFS, root)

	iSync.ProcessFiles()

	wantDirs := map[string]bool{"notes": true, "notes/my": true}
	if !reflect.DeepEqual(iSync.Dirs, wantDirs) {
		t.Errorf("got %v, want %v", iSync.Dirs, wantDirs)
	}

	wantFiles := map[string][]string{
		"notes/my/note.md":  {"notes/my/assets/image01.png", "notes/my/assets/image02.png"},
		"notes/my/note2.md": {"notes/my/assets/image02.png"},
	}
	if !reflect.DeepEqual(iSync.Files, wantFiles) {
		t.Errorf("got %v, want %v", iSync.Files, wantFiles)
	}

	wantImages := map[string][]string{
		"notes/my/assets/image01.png": {"notes/my/note.md"},
		"notes/my/assets/image02.png": {"notes/my/note.md", "notes/my/note2.md"},
	}

	if !reflect.DeepEqual(iSync.Images, wantImages) {
		t.Errorf("got %v, want %v", iSync.Images, wantImages)
	}
}
