package fswatcher

import (
	"errors"
	"io/fs"
	"os"
	"reflect"
	"testing"
	"testing/fstest"
)

func TestAdd(t *testing.T) {
	t.Run("add files", func(t *testing.T) {
		var err error

		files := []string{"path/note.md", "path/some_dir"}

		ff := map[string]*fstest.MapFile{}

		for _, v := range files {
			ff[v] = &fstest.MapFile{
				Data: nil,
			}
		}

		fsys := fstest.MapFS(ff)
		p := NewFsPoller(fsys)

		want := map[string]os.FileInfo{}

		for _, v := range files {
			p.Add(v)
			want[v], err = fsys.Stat(v)
			if err != nil {
				t.Error(err)
			}
		}

		if !reflect.DeepEqual(want, p.files) {
			t.Errorf("length: want %s, got %s", want, p.files)
		}

	})

	t.Run("emit error if closed", func(t *testing.T) {
		p := NewFsPoller(fstest.MapFS{})
		p.closed = true
		err := p.Add("file")

		want := errors.New("poller is closed")
		if err == nil || err.Error() != want.Error() {
			t.Errorf("Should throw error. got %s, want %s", err, want)
		}
	})

	t.Run("file is not exist", func(t *testing.T) {
		p := NewFsPoller(fstest.MapFS{})
		err := p.Add("some_folder")
		if !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("Should throw error. %s", err)
		}
	})
}
