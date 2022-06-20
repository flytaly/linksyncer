package imagesync

import (
	"path/filepath"
	"reflect"
	"testing"
	"testing/fstest"
)

func TestProcessFiles(t *testing.T) {
	root := "/home/user/notes"

	j := filepath.Join

	markdown := `
				![alt text](./assets/image01.png)
				![alt text](./assets/image02.png)
	`
	mapFS := fstest.MapFS{
		"n/sub/note.md":  {Data: []byte(markdown)},
		"n/sub/note2.md": {Data: []byte("![alt text](./assets/image02.png)")},
	}

	iSync := New(mapFS, root)

	iSync.ProcessFiles()

	wantDirs := map[string]bool{
		j(root, "n"):     true,
		j(root, "n/sub"): true,
	}
	if !reflect.DeepEqual(iSync.Dirs, wantDirs) {
		t.Errorf("got %v, want %v", iSync.Dirs, wantDirs)
	}

	wantFiles := map[string][]string{
		j(root, "n/sub/note.md"):  {j(root, "n/sub/assets/image01.png"), j(root, "n/sub/assets/image02.png")},
		j(root, "n/sub/note2.md"): {j(root, "n/sub/assets/image02.png")},
	}
	if !reflect.DeepEqual(iSync.Files, wantFiles) {
		t.Errorf("got %v, want %v", iSync.Files, wantFiles)
	}

	wantImages := map[string][]string{
		j(root, "n/sub/assets/image01.png"): {j(root, "n/sub/note.md")},
		j(root, "n/sub/assets/image02.png"): {j(root, "n/sub/note.md"), j(root, "n/sub/note2.md")},
	}

	if !reflect.DeepEqual(iSync.Images, wantImages) {
		t.Errorf("got %v, want %v", iSync.Images, wantImages)
	}
}
