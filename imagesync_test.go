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

	wantFiles := map[string][]ImageInfo{
		j(root, "n/sub/note.md"): {
			ImageInfo{
				absPath:  j(root, "n/sub/", "./assets/image01.png"),
				original: "./assets/image01.png",
			},
			ImageInfo{
				absPath:  j(root, "n/sub/", "./assets/image02.png"),
				original: "./assets/image02.png",
			},
		},
		j(root, "n/sub/note2.md"): {
			ImageInfo{
				absPath:  j(root, "n/sub/", "./assets/image02.png"),
				original: "./assets/image02.png",
			},
		},
	}
	if !reflect.DeepEqual(iSync.Files, wantFiles) {
		t.Errorf("got %v,\n want %v", iSync.Files, wantFiles)
	}

	wantImages := map[string][]string{
		j(root, "n/sub/assets/image01.png"): {j(root, "n/sub/note.md")},
		j(root, "n/sub/assets/image02.png"): {j(root, "n/sub/note.md"), j(root, "n/sub/note2.md")},
	}

	if !reflect.DeepEqual(iSync.Images, wantImages) {
		t.Errorf("got %v, want %v", iSync.Images, wantImages)
	}
}

func TestRemoveFile(t *testing.T) {
	root := "/home/user/notes"
	j := filepath.Join

	note1 := j(root, "/note1.md")
	note2 := j(root, "/note2.md")
	img1 := ImageInfo{absPath: j(root, "./assets/i1.png"), original: "./assets/i1.png"}
	img2 := ImageInfo{absPath: j(root, "./assets/i2.png"), original: "./assets/i2.png"}
	files := map[string][]ImageInfo{note1: {img1, img2}, note2: {img2}}
	images := map[string][]string{
		j(root, "assets/i1.png"): {note1},
		j(root, "assets/i2.png"): {note1, note2},
	}

	t.Run("remove note 1", func(t *testing.T) {
		iSync := New(fstest.MapFS{}, root)

		for k, v := range files {
			iSync.Files[k] = v
		}
		for k, v := range images {
			iSync.Images[k] = v
		}

		iSync.RemoveFile(note1)

		wantFiles := map[string][]ImageInfo{note2: {img2}}
		if !reflect.DeepEqual(iSync.Files, wantFiles) {
			t.Errorf("got %v, want %v", iSync.Files, wantFiles)
		}

		wantImages := map[string][]string{
			j(root, "assets/i1.png"): nil,
			j(root, "assets/i2.png"): {note2},
		}
		if !reflect.DeepEqual(iSync.Images, wantImages) {
			t.Errorf("got %v, want %v", iSync.Images, wantImages)
		}
	})

	t.Run("remove note 2", func(t *testing.T) {
		iSync := New(fstest.MapFS{}, root)
		for k, v := range files {
			iSync.Files[k] = v
		}
		for k, v := range images {
			iSync.Images[k] = v
		}

		iSync.RemoveFile(note2)

		wantFiles := map[string][]ImageInfo{note1: {img1, img2}}
		if !reflect.DeepEqual(iSync.Files, wantFiles) {
			t.Errorf("got %v, want %v", iSync.Files, wantFiles)
		}

		wantImages := map[string][]string{
			j(root, "assets/i1.png"): {note1},
			j(root, "assets/i2.png"): {note1},
		}
		if !reflect.DeepEqual(iSync.Images, wantImages) {
			t.Errorf("got %v, want %v", iSync.Images, wantImages)
		}
	})

}
