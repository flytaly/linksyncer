package fswatcher

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/flytaly/imagesync/testutils"
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
		ff[v] = &fstest.MapFile{Data: []byte(fmt.Sprintf("file content: %s", v))}
	}
	return ff
}

func TestAdd(t *testing.T) {
	t.Run("add files", func(t *testing.T) {
		root := j("path", "notes")

		fileList := []string{
			root,
			j(root, "note.md"),
			j(root, "some_dir"),
			j(root, "some_dir", "note2.md"),
		}

		fsys := createFS(fileList)

		watches := map[string]struct{}{root: {}}

		p := makePoller(fsys, root)
		_, err := p.Add(root)
		failIfErr(t, err)

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
		_, err := p.Add("path1")
		failIfErr(t, err)
		_, err = p.Add("path2")
		failIfErr(t, err)

		err = p.Remove("path1")
		failIfErr(t, err)
		_, err = fsys.Stat("path2")
		failIfErr(t, err)
		assert.Contains(t, p.files, "path2")
		assert.NotContains(t, p.files, "path1")
	})
}

func TestClose(t *testing.T) {
	fsys := createFS([]string{"some_path"})
	p := makePoller(fsys, ".")

	go func() {
		failIfErr(t, p.Start(time.Second))
	}()

	time.Sleep(time.Millisecond)
	err := p.Close()
	failIfErr(t, err)
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
		_, err := p.Add(".")
		failIfErr(t, err)

		newFiles := map[string]string{ // path => event's filename
			"newFile1.txt":       "newFile1.txt",
			"newFile2.txt":       "newFile2.txt",
			"newFolder/":         "newFolder",
			"newFolder/file.txt": "newFolder/file.txt",
		}

		evs := map[string]Event{}
		for _, name := range newFiles {
			evs[name] = Event{Op: Create, Name: name}
		}
		ExpectEvents(t, p, minWait, evs)

		go func() {
			failIfErr(t, p.Start(0))
		}()

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
		_, err := p.Add(".")
		failIfErr(t, err)

		remove := []string{"file2.txt"}

		evs := map[string]Event{}
		for _, name := range remove {
			evs[name] = Event{Op: Remove, Name: name}
		}
		ExpectEvents(t, p, minWait, evs)

		go func() {
			failIfErr(t, p.Start(0))
		}()

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
		tempFolder, tempFile := "temp", j("temp", "file2.png")
		fsys := createFS([]string{
			j("folder", "file1.png"),
			tempFile,
		})
		p := makePoller(fsys, ".")
		_, err := p.Add("folder")
		failIfErr(t, err)
		_, err = p.Add(tempFolder)
		failIfErr(t, err)

		assert.Contains(t, p.watches, "folder")
		assert.Contains(t, p.watches, tempFolder)

		evs := map[string]Event{}
		evs[tempFolder] = Event{Op: Remove, Name: "temp"}
		evs[tempFile] = Event{Op: Remove, Name: tempFile}
		ExpectEvents(t, p, minWait, evs)

		go func() {
			failIfErr(t, p.Start(0))
		}()

		go func() {
			time.Sleep(time.Millisecond * 2)
			delete(fsys, tempFile)
			delete(fsys, tempFolder)
		}()

		<-p.done
		assert.NotContainsf(t, p.files, tempFile, "shouldn't contain removed path")
		assert.NotContainsf(t, p.watches, tempFile, "shouldn't watch removed path")

		assert.NotContainsf(t, p.files, j("temp", "file2.png"), "shouldn't contain removed file")
	})

	t.Run("RENAME", func(t *testing.T) {
		fsys := createFS([]string{"file1.txt", "file2.txt", j("folder/"), j("folder", "file3.txt")})
		p := makePoller(fsys, ".")
		_, err := p.Add(".")
		failIfErr(t, err)

		rename := map[string]string{
			"file2.txt":           "renamed.txt",
			j("folder/file3.txt"): "renamed2.txt",
		}

		evs := map[string]Event{}
		for from, to := range rename {
			evs[from] = Event{Op: Rename, Name: from, NewPath: to}
		}
		ExpectEvents(t, p, minWait, evs)

		go func() {
			failIfErr(t, p.Start(0))
		}()

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
		dir1 := "temp"
		dir2 := "another_dir"
		moveFrom := j(dir1, "file1.png")
		moveTo := j(dir2, dir1, "file1.png")
		fsys := createFS([]string{moveFrom, dir2})
		p := makePoller(fsys, ".")
		_, err := p.Add(dir1)
		failIfErr(t, err)
		_, err = p.Add(dir2)
		failIfErr(t, err)

		evs := map[string]Event{}
		evs[dir1] = Event{Op: Rename, Name: dir1, NewPath: j(dir2, dir1)}
		evs[moveFrom] = Event{Op: Rename, Name: moveFrom, NewPath: moveTo}
		ExpectEvents(t, p, minWait, evs)

		go func() {
			failIfErr(t, p.Start(0))
		}()

		go func() {
			time.Sleep(time.Millisecond * 2)
			fsys[moveTo] = fsys[moveFrom]
			delete(fsys, moveFrom)
			delete(fsys, dir1)
		}()

		<-p.done
		assert.NotContainsf(t, p.files, dir1, "shouldn't contain previous path")
		assert.NotContainsf(t, p.files, moveFrom, "shouldn't contain previous path")
		assert.NotContainsf(t, p.watches, dir1, "shouldn't watch removed path")
	})

	t.Run("WRITE", func(t *testing.T) {
		fsys := createFS([]string{"file1.txt", "file2.txt"})
		p := makePoller(fsys, ".")
		_, err := p.Add(".")
		failIfErr(t, err)

		write := []string{"file2.txt"}

		evs := map[string]Event{}
		for _, name := range write {
			evs[name] = Event{Op: Write, Name: name}
		}
		ExpectEvents(t, p, minWait, evs)

		go func() {
			failIfErr(t, p.Start(0))
		}()

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
		root := "folder"
		skip := []string{j(root, "node_modules"), j(root, "movie.mp4"), j(root, "skip.txt")}
		noskip := []string{j(root, "note.md")}
		fsys := createFS(noskip)
		for _, f := range skip {
			fsys[f] = &fstest.MapFile{}
		}
		p := makePoller(fsys, root)
		p.AddShouldSkipHook(func(fi fs.FileInfo) bool {
			return !fi.IsDir() && filepath.Ext(fi.Name()) != ".md"
		})

		_, err := p.Add(root)
		failIfErr(t, err)

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
		go func() {
			err := p.Close()
			if err != nil {
				t.Errorf("watcher close error: %s", err)
			}
		}()
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
				go func() {
					err := p.Close()
					if err != nil {
						t.Errorf("watcher close error: %s", err)
					}
				}()
			case <-p.scanDone:
				return
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
