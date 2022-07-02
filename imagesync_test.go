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
				absPath:      j(root, "n/sub/", "./assets/image01.png"),
				originalLink: "./assets/image01.png",
			},
			ImageInfo{
				absPath:      j(root, "n/sub/", "./assets/image02.png"),
				originalLink: "./assets/image02.png",
			},
		},
		j(root, "n/sub/note2.md"): {
			ImageInfo{
				absPath:      j(root, "n/sub/", "./assets/image02.png"),
				originalLink: "./assets/image02.png",
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
	img1 := ImageInfo{absPath: j(root, "./assets/i1.png"), originalLink: "./assets/i1.png"}
	img2 := ImageInfo{absPath: j(root, "./assets/i2.png"), originalLink: "./assets/i2.png"}
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
			// j(root, "assets/i1.png"): nil,
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

func TestRenameFile(t *testing.T) {
	root := "/home/user/notes"
	j := filepath.Join

	prevName := j(root, "/original_name.md")
	newName := j(root, "/new_name.md")
	note2 := j(root, "/note2.md")
	img1 := ImageInfo{absPath: j(root, "./assets/i1.png"), originalLink: "./assets/i1.png"}
	img2 := ImageInfo{absPath: j(root, "./assets/i2.png"), originalLink: "./assets/i2.png"}

	iSync := New(fstest.MapFS{
		"new_name.md": {Data: []byte("")},
	}, root)

	iSync.Files = map[string][]ImageInfo{prevName: {img1, img2}, note2: {img2}}
	iSync.Images = map[string][]string{
		j(root, "assets/i1.png"): {prevName},
		j(root, "assets/i2.png"): {prevName, note2},
	}

	extractImagesOriginal := extractImages

	extractImages = func(filePath, content string) []ImageInfo {
		return []ImageInfo{img1, img2}
	}

	iSync.RenameFile(prevName, newName)

	extractImages = extractImagesOriginal

	wantFiles := map[string][]ImageInfo{newName: {img1, img2}, note2: {img2}}
	if !reflect.DeepEqual(iSync.Files, wantFiles) {
		t.Errorf("got %v, want %v", iSync.Files, wantFiles)
	}

	wantImages := map[string][]string{
		j(root, "assets/i1.png"): {newName},
		j(root, "assets/i2.png"): {note2, newName},
	}
	if !reflect.DeepEqual(iSync.Images, wantImages) {
		t.Errorf("got %v, want %v", iSync.Images, wantImages)
	}
}

func TestUpdateImageLinks(t *testing.T) {
	root := "/home/user/notes"
	j := filepath.Join
	filePath := j(root, "my_note.md")

	imgs := []RenamedImage{{
		prevPath: j(root, "./assets/image01.png"),
		newPath:  j(root, "./images/image01.png"),
		link:     "./assets/image01.png",
	}}

	content := []byte("![alt text](" + imgs[0].link + ")")

	mapFS := fstest.MapFS{"my_note.md": {Data: content}}

	fileWriterOriginal := writeFile
	t.Cleanup(func() {
		writeFile = fileWriterOriginal
	})

	t.Run("update image links in the file", func(t *testing.T) {
		iSync := New(mapFS, root)
		iSync.AddFile(filePath)
		writtenData := ""

		writeFile = func(fPath string, data []byte) error {
			writtenData = string(data)
			return nil
		}

		err := iSync.UpdateImageLinks(filePath, imgs)

		if err != nil {
			t.Error(err)
		}

		want := "![alt text](images/image01.png)"

		if writtenData != want {
			t.Errorf("expect %s, got %s", want, writtenData)
		}
	})

	t.Run("update images in the imagesync struct", func(t *testing.T) {
		iSync := New(mapFS, root)
		iSync.AddFile(filePath)

		writeFile = func(fPath string, data []byte) error {
			return nil
		}

		err := iSync.UpdateImageLinks(filePath, imgs)
		if err != nil {
			t.Error(err)
		}

		updatedImageList := map[string][]string{}

		for _, imgs := range imgs {
			updatedImageList[imgs.newPath] = []string{filePath}
		}

		if !reflect.DeepEqual(updatedImageList, iSync.Images) {
			t.Errorf("want %v, got %v", updatedImageList, iSync.Images)
		}
	})
}
