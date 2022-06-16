package imagesync

import (
	"imagesync/testutils"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestWatchList(t *testing.T) {
	emptyFile := []byte("")
	j := filepath.Join

	files := []string{
		j("notes", "note.md"),
		j("pages", "subfolder", "page.html"),
		j("somepage.htm"),
	}

	skipFiles := []string{
		j("skip", "text.txt"),
		j("skip", "video.mp4"),
		j(".git", "f.md"),
		j("node_modules", "module", "page.html"),
	}

	dirs := []string{
		j("notes"),
		j("pages"),
		j("pages", "subfolder"),
		j("skip"),
	}

	fs := fstest.MapFS{
		files[0]: {Data: emptyFile},
		files[1]: {Data: emptyFile},
		files[2]: {Data: emptyFile},
	}

	for _, v := range skipFiles {
		fs[v] = &fstest.MapFile{Data: []byte("")}
	}

	gotDirs, gotFiles, err := WatchList(fs, ".")

	if err != nil {
		t.Fatal(err)
	}

	if d := testutils.Difference(gotDirs, dirs); len(d) > 0 {
		t.Errorf("Directories: got %+v, want %+v", gotDirs, dirs)
		t.Errorf("difference %+v", d)
	}

	if d := testutils.Difference(gotFiles, files); len(d) > 0 {
		t.Errorf("Files: got %+v, want %+v", gotFiles, files)
		t.Errorf("difference %+v", d)
	}
}
