package fswatcher

import (
	"errors"
	"imagesync/testutils"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"testing/fstest"
	"time"
)

func makePoller(fsys fs.FS, root string) *fsPoller {
	return &fsPoller{
		events:  make(chan Event),
		errors:  make(chan error),
		closed:  false,
		done:    make(chan struct{}),
		fsys:    fsys,
		mu:      new(sync.Mutex),
		root:    root,
		files:   make(map[string]os.FileInfo),
		watches: map[string]struct{}{},
	}
}

var j = path.Join

// return fs with empty files and folders from a slice with filenames
func createFS(files []string) fstest.MapFS {
	var ff fstest.MapFS = make(map[string]*fstest.MapFile)
	for _, v := range files {
		if filepath.Ext(v) == "" { // is dir
			ff[v] = &fstest.MapFile{Mode: os.ModeDir}
			continue
		}
		ff[v] = &fstest.MapFile{}
	}
	return ff
}

func TestAdd(t *testing.T) {
	t.Run("add files", func(t *testing.T) {
		root := "path"

		fileList := []string{
			j(root, "notes"),
			j(root, "notes", "note.md"),
			j(root, "notes", "some_dir"),
			j(root, "notes", "some_dir", "note2.md"),
			j(root, "notes", "ignored_dir"),
		}

		fsys := createFS(fileList)
		fsys[j(root, "notes", "ignored_dir", "file1.md")] = &fstest.MapFile{}

		watches := map[string]struct{}{
			j(root, "notes"):             {},
			j(root, "notes", "some_dir"): {},
		}

		p := makePoller(fsys, ".")

		for name := range watches {
			failIfErr(t, p.Add(name))
		}

		if !reflect.DeepEqual(p.watches, watches) {
			t.Errorf("watches: got %v, want %v", p.watches, watches)
		}

		testutils.CompareMapKeys(t, p.files, fileList)

	})

	t.Run("emit error if closed", func(t *testing.T) {
		p := makePoller(fstest.MapFS{}, ".")
		p.closed = true
		err := p.Add("file")

		want := errors.New("poller is closed")
		if err == nil || err.Error() != want.Error() {
			t.Errorf("Should throw error. got %s, want %s", err, want)
		}
	})

	t.Run("file is not exist", func(t *testing.T) {
		p := makePoller(fstest.MapFS{}, ".")
		err := p.Add("some_folder")
		if !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("Should throw error. %s", err)
		}
	})
}

func TestRemove(t *testing.T) {
	t.Run("remove file", func(t *testing.T) {
		fsys := createFS([]string{"path1", "path2"})
		p := makePoller(fsys, ".")
		failIfErr(t, p.Add("path1"))
		failIfErr(t, p.Add("path2"))
		failIfErr(t, p.Remove("path1"))
		stat, err := fsys.Stat("path2")
		failIfErr(t, err)
		want := map[string]os.FileInfo{"path2": stat}
		if !reflect.DeepEqual(p.files, want) {
			t.Errorf("got %s, want %s", p.files, want)
		}
	})
}

func TestClose(t *testing.T) {
	fsys := createFS([]string{"some_path"})
	p := makePoller(fsys, ".")
	go p.Start(time.Second)

	time.Sleep(time.Millisecond)
	p.Close()
	select {
	case <-time.After(time.Millisecond):
		t.Error("'done' should be closed")
	case <-p.done:
		if !p.closed {
			t.Errorf(`"closed" should be "true"`)
		}
	}

}

func failIfErr(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}
