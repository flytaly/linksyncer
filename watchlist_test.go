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

	watchFiles := []string{
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

	want := []string{
		j("notes"),
		j("pages"),
		j("pages", "subfolder"),
		j("skip"),
	}

	want = append(want, watchFiles...)

	fs := fstest.MapFS{
		watchFiles[0]: {Data: emptyFile},
		watchFiles[1]: {Data: emptyFile},
		watchFiles[2]: {Data: emptyFile},
	}

	for _, v := range skipFiles {
		fs[v] = &fstest.MapFile{Data: []byte("")}
	}

	watchPaths, err := WatchList(fs, ".")

	if err != nil {
		t.Fatal(err)
	}

	got := make([]string, 0, len(watchPaths))

	for path := range watchPaths {
		got = append(got, path)
	}

	if d := testutils.Difference(got, want); len(d) > 0 {
		t.Errorf("got %+v, want %+v", got, want)
		t.Errorf("difference %+v", d)
	}
}
