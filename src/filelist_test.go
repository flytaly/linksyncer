package imagesync

import (
	"path/filepath"
	"reflect"
	"testing"
	"testing/fstest"
)

func TestFileList(t *testing.T) {
	emptyFile := []byte("")

	want := []string{
		filepath.Join("notes", "note.md"),
		filepath.Join("page.htm"),
		filepath.Join("pages", "page.html"),
	}

	skipFiles := []string{
		"skip/text.txt",
		"skip/video.mp4",
		filepath.Join(".git/f.md"),
		filepath.Join("node_modules/module/page.html"),
	}

	fs := fstest.MapFS{
		want[0]: {Data: emptyFile},
		want[1]: {Data: emptyFile},
		want[2]: {Data: emptyFile},
	}

	for _, v := range skipFiles {
		fs[v] = &fstest.MapFile{Data: []byte("")}
	}

	got, err := FileList(fs, ".")

	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}
