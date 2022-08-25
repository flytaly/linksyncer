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
		events: make(chan Event),
		errors: make(chan error),
		closed: false,
		done:   make(chan struct{}),
		fsys:   fsys,
		mu:     new(sync.Mutex),
		root:   root,
		files:  make(map[string]os.FileInfo),
	}
}

var j = path.Join

func TestAdd(t *testing.T) {
	t.Run("add files", func(t *testing.T) {
		root := "path"

		files := []string{
			j(root, "note.md"),
			j(root, "some_dir/note2.md"),
		}

		ff := map[string]*fstest.MapFile{}

		for _, v := range files {
			ff[v] = &fstest.MapFile{
				Data: nil,
			}
		}

		fsys := fstest.MapFS(ff)
		p := makePoller(fsys, root)

		err := p.Add(j(root, "path"))
		if err != nil {
			t.Error(err)
		}

		want := []string{root, j(root, "note.md"), j(root, "some_dir")}
		got := []string{}
		for name := range p.files {
			got = append(got, name)
		}

		testutils.Compare(t, got, want)
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
		fsys := fstest.MapFS{
			"path1": {Data: []byte("")},
			"path2": {Data: []byte("")},
		}
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
	fsys := fstest.MapFS{"some_path": {Data: []byte("")}}
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
