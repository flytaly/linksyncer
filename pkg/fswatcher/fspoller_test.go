package fswatcher

import (
	"errors"
	"imagesync/testutils"
	"io/fs"
	"os"
	"path"
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

// return fs with empty files from a slice with filenames
func createFS(files []string) *fstest.MapFS {
	ff := map[string]*fstest.MapFile{}
	for _, v := range files {
		ff[v] = &fstest.MapFile{Data: nil}
	}
	fsys := fstest.MapFS(ff)

	return &fsys
}

func TestAdd(t *testing.T) {
	t.Run("add files", func(t *testing.T) {
		root := "path"

		fsys := createFS([]string{
			j(root, "notes", "note.md"),
			j(root, "notes", "some_dir", "note2.md"),
			j(root, "notes", "ignored_dir", "ignored.md"),
		})

		fileList := []string{
			j(root, "notes"),
			j(root, "notes", "note.md"),
			j(root, "notes", "some_dir"),
			j(root, "notes", "some_dir", "note2.md"),
			j(root, "notes", "ignored_dir"),
		}

		watches := map[string]struct{}{
			j(root, "notes"):             {},
			j(root, "notes", "some_dir"): {},
		}

		p := makePoller(fsys, ".")

		for name := range watches {
			err := p.Add(name)
			if err != nil {
				t.Error(err)
			}
		}

		got := []string{}
		for name := range p.files {
			got = append(got, name)
		}

		testutils.Compare(t, got, fileList)

		if !reflect.DeepEqual(p.watches, watches) {
			t.Errorf("watches: got %v, want %v", p.watches, watches)
		}
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
		p.Add("path1")
		p.Add("path2")
		p.Remove("path1")
		stat, _ := fsys.Stat("path2")
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
