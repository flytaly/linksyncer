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

	"github.com/stretchr/testify/assert"
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
		files:   make(map[string]*os.FileInfo),
		watches: map[string]struct{}{},
	}
}

var minWait = MIN_INTERVAL + time.Millisecond*5

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
			_, err := p.Add(name)
			failIfErr(t, err)
		}

		if !reflect.DeepEqual(p.watches, watches) {
			t.Errorf("watches: got %v, want %v", p.watches, watches)
		}

		testutils.CompareMapKeys(t, p.files, fileList)
	})

	t.Run("emit error if closed", func(t *testing.T) {
		p := makePoller(fstest.MapFS{}, ".")
		p.closed = true
		_, err := p.Add("file")

		want := errors.New("poller is closed")
		if err == nil || err.Error() != want.Error() {
			t.Errorf("Should throw error. got %s, want %s", err, want)
		}
	})

	t.Run("file is not exist", func(t *testing.T) {
		p := makePoller(fstest.MapFS{}, ".")
		_, err := p.Add("some_folder")
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
		_, err := fsys.Stat("path2")
		failIfErr(t, err)
		assert.Contains(t, p.files, "path2")
		assert.NotContains(t, p.files, "path1")
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

func TestEvent(t *testing.T) {

	t.Run("CREATE", func(t *testing.T) {
		fsys := createFS([]string{"file"})
		p := makePoller(fsys, ".")
		p.Add(".")

		newFiles := map[string]string{ // path => event's filename
			"newFile1.txt":       "newFile1.txt",
			"newFile2.txt":       "newFile2.txt",
			"newFolder/file.txt": "newFolder",
		}

		evs := map[string]Event{}
		for _, name := range newFiles {
			evs[name] = Event{Op: Create, Name: name}
		}
		ExpectEvents(t, p, minWait, evs)

		go p.Start(0)

		go func() {
			time.Sleep(time.Millisecond * 2)
			for path := range newFiles {
				fsys[path] = &fstest.MapFile{}
			}
		}()

		<-p.done

		for _, f := range newFiles {
			assert.Contains(t, p.files, f, "should contain path: %s", f)
		}
	})

	t.Run("REMOVE", func(t *testing.T) {
		fsys := createFS([]string{"file1.txt", "file2.txt", "file3.txt"})
		p := makePoller(fsys, ".")
		p.Add(".")

		remove := []string{"file2.txt"}

		evs := map[string]Event{}
		for _, name := range remove {
			evs[name] = Event{Op: Remove, Name: name}
		}
		ExpectEvents(t, p, minWait, evs)

		go p.Start(0)

		go func() {
			time.Sleep(time.Millisecond * 2)
			for _, path := range remove {
				delete(fsys, path)
			}
		}()

		<-p.done

		for _, f := range remove {
			assert.NotContains(t, p.files, f, "shouldn't contain removed path: %s", f)
		}
	})

	t.Run("REMOVE watched path", func(t *testing.T) {
		fsys := createFS([]string{
			j("folder1", "file1.png"),
			j("temp", "file2.png"),
			"tempFile.txt",
		})
		pathToRemove := []string{"tempFile.txt", "temp"}
		p := makePoller(fsys, ".")
		p.Add(".")

		evs := map[string]Event{}
		for _, f := range pathToRemove {
			p.Add(f)
			evs[f] = Event{Op: Remove, Name: f}
		}
		ExpectEvents(t, p, minWait, evs)

		go p.Start(0)

		go func() {
			time.Sleep(time.Millisecond * 2)
			delete(fsys, j("temp", "file2.png"))
			for _, f := range pathToRemove {
				delete(fsys, f)
			}
		}()

		<-p.done
		for _, f := range pathToRemove {
			assert.NotContainsf(t, p.files, f, "shouldn't contain removed path")
			assert.NotContainsf(t, p.watches, f, "shouldn't watch removed path")
		}

		assert.NotContainsf(t, p.files, j("temp", "file2.png"), "shouldn't contain removed file")
	})

	t.Run("RENAME", func(t *testing.T) {
		fsys := createFS([]string{"file1.txt", "file2.txt"})
		p := makePoller(fsys, ".")
		p.Add(".")

		rename := map[string]string{"file2.txt": "renamed.txt"}

		evs := map[string]Event{}
		for from, to := range rename {
			evs[from] = Event{Op: Rename, Name: from, NewPath: to}
		}
		ExpectEvents(t, p, minWait, evs)

		go p.Start(0)

		go func() {
			time.Sleep(time.Millisecond * 2)
			for from, to := range rename {
				fsys[to] = fsys[from]
				delete(fsys, from)
			}
		}()

		<-p.done

		for from, to := range rename {
			assert.NotContains(t, p.files, from, "shouldn't contain removed path")
			assert.Contains(t, p.files, to, "should contain renamed path")
		}
	})

	t.Run("RENAME watched path", func(t *testing.T) {
		dirFrom := "temp"
		dirTo := "renamed"
		fileFrom := j(dirFrom, "file.png")
		fileTo := j(dirTo, "file.png")
		fsys := createFS([]string{fileFrom})
		p := makePoller(fsys, ".")
		p.Add(".")
		p.Add(dirFrom)

		evs := map[string]Event{}
		evs[dirFrom] = Event{Op: Rename, Name: dirFrom, NewPath: dirTo}
		ExpectEvents(t, p, minWait, evs)

		go p.Start(0)

		go func() {
			time.Sleep(time.Millisecond * 2)
			fsys[fileTo] = fsys[fileFrom]
			delete(fsys, fileFrom)
		}()

		<-p.done
		assert.NotContainsf(t, p.files, dirFrom, "shouldn't contain previous path")
		assert.NotContainsf(t, p.files, fileFrom, "shouldn't contain previous path")
		assert.NotContainsf(t, p.watches, dirFrom, "shouldn't watch removed path")
	})

	t.Run("WRITE", func(t *testing.T) {
		fsys := createFS([]string{"file1.txt", "file2.txt"})
		p := makePoller(fsys, ".")
		p.Add(".")

		write := []string{"file2.txt"}

		evs := map[string]Event{}
		for _, name := range write {
			evs[name] = Event{Op: Write, Name: name}
		}
		ExpectEvents(t, p, minWait, evs)

		go p.Start(0)

		go func() {
			time.Sleep(time.Millisecond * 2)
			for _, path := range write {
				fsys[path] = &fstest.MapFile{} // create new reference
				fsys[path].ModTime = fsys[path].ModTime.Add(time.Second)
			}
		}()

		<-p.done

		for _, name := range write {
			assert.Contains(t, p.files, name, "should contain renamed path")
		}
	})
}

func TestShouldSkipHook(t *testing.T) {
	t.Run("skip files", func(t *testing.T) {
		skip := []string{"node_modules", "movie.mp4", "skip.txt"}
		noskip := []string{"note.md"}
		fsys := createFS(noskip)
		for _, f := range skip {
			fsys[f] = &fstest.MapFile{}
		}
		p := makePoller(fsys, ".")
		p.AddShouldSkipHook(func(fi fs.FileInfo) bool {
			return filepath.Ext(fi.Name()) != ".md"
		})
		p.Add(".")

		for _, f := range skip {
			assert.NotContains(t, p.files, f)
		}
		for _, f := range noskip {
			assert.Contains(t, p.files, f)
		}
	})
}

func ExpectEvents(t *testing.T, p *fsPoller, await time.Duration, want map[string]Event) {
	gotEvents := map[string]Event{}

	check := func() {
		assert.Equal(t, want, gotEvents, "should trigger events")
		go p.Close()
	}

	go func() {
		for {
			select {
			case event := <-p.Events():
				gotEvents[event.Name] = event
				if len(want) == len(gotEvents) {
					check()
				}
			case err := <-p.Errors():
				t.Errorf("watcher error event: %s", err)
				go p.Close()
			case <-p.done:
				return
			}
		}
	}()

	go func() {
		time.Sleep(await)
		if !p.closed {
			t.Errorf("Events were not triggered in time")
			check()
		}
	}()
}

func failIfErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
