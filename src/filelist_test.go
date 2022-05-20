package imagesync

import (
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
	"testing/fstest"
)

func TestFileList(t *testing.T) {
	want := []string{
		filepath.Join("notes", "note.md"),
		filepath.Join("pages", "page.html"),
	}

	emptyFile := []byte("")

	fs := fstest.MapFS{
		want[0]:                    {Data: emptyFile},
		want[1]:                    {Data: emptyFile},
		filepath.Join(".git/f.md"): {Data: emptyFile},
		filepath.Join("node_modules/module/page.html"): {Data: emptyFile},
	}

	got, err := FileList(fs, ".")

	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}

	fmt.Println(got)
}
