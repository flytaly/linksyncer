package fswatcher

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sync"
	"time"
)

const MIN_INTERVAL = time.Millisecond * 20

// fsPoller is polling implementation of FileWatcher interface
type fsPoller struct {
	// watched files and dirs
	watches map[string]struct{}
	// stores info about files and dirs inside watched paths
	files      map[string]*fs.FileInfo
	events     chan Event
	errors     chan error
	done       chan struct{}
	scanDone   chan struct{}
	shouldSkip func(fs.FileInfo) bool
	fsys       fs.FS
	// path to the root directory
	root    string
	running bool

	mu     *sync.Mutex
	closed bool
}

func (p *fsPoller) AddShouldSkipHook(fn func(fs.FileInfo) bool) {
	p.shouldSkip = fn
}

// Add adds given name into the list of the watched paths.
// If name is a directory, then retrieves FileInfo of nested files, saves them
// and returns.
func (p *fsPoller) Add(name string) (map[string]*fs.FileInfo, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil, errors.New("poller is closed")
	}

	var err error
	relativePath := name

	if filepath.IsAbs(relativePath) {
		relativePath, err = filepath.Rel(p.root, name)
		if err != nil {
			return nil, err
		}
	}

	list, err := p.listDirFiles(relativePath, true)
	if err != nil /* && errors.Is(err, fs.ErrNotExist) */ {
		return nil, err
	}

	for fname, fi := range list {
		p.files[fname] = fi
	}

	p.watches[relativePath] = struct{}{}

	return list, nil
}

// listDirFiles returns list of the files if name is a directory,
// if name isn't a directory then just returns it's FileInfo.
func (p *fsPoller) listDirFiles(name string, recursively bool) (map[string]*fs.FileInfo, error) {
	files := map[string]*fs.FileInfo{}

	fInfo, err := fs.Stat(p.fsys, name)
	if err != nil {
		return nil, err
	}
	files[name] = &fInfo

	if !fInfo.IsDir() {
		return files, nil
	}

	err = fs.WalkDir(p.fsys, name, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		stat, err := d.Info()
		if err != nil {
			return err
		}
		if p.shouldSkip != nil && p.shouldSkip(stat) { // skip
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		files[path] = &stat
		if d.IsDir() && path != name && !recursively {
			return filepath.SkipDir
		}
		return nil
	})

	if err != nil {
		fmt.Println(err)
	}

	return files, nil
}

// scanForChanges checks folders for changes
func (p *fsPoller) scanForChanges() {
	p.mu.Lock()
	defer p.mu.Unlock()

	addedFiles := map[string]*fs.FileInfo{}
	updated := map[string]*fs.FileInfo{}
	renamed := map[string]*fs.FileInfo{}

	for path := range p.watches {
		files, err := p.listDirFiles(path, true)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				delete(p.watches, path)
				continue
			}
			p.errors <- err
			continue
		}
		p.separate(files, addedFiles, updated)
	}

	removed := []string{}

	for path, oldInfo := range p.files {
		newInfo, existed := updated[path]
		if existed {
			p.onFileWrite(path, oldInfo, newInfo)
			continue
		}
		p.onFileRemove(path, addedFiles, oldInfo, renamed)
		removed = append(removed, path)
	}

	for _, path := range removed {
		delete(p.files, path)
	}

	for path, fi := range updated {
		p.files[path] = fi
	}

	for name, info := range addedFiles {
		p.files[name] = info
		if _, ok := renamed[name]; !ok {
			err := p.SendEvent(Event{Op: Create, Name: name})
			if err != nil {
				p.errors <- err
			}
		}
	}
}

// onFileWrite checks if file with given path was changed, if positive triggers Write event
func (p *fsPoller) onFileWrite(path string, oldFi, newFi *fs.FileInfo) bool {
	if (*oldFi).ModTime() != (*newFi).ModTime() {
		err := p.SendEvent(Event{Op: Write, Name: path})
		if err != nil {
			p.errors <- err
		}
		return true
	}
	return false
}

// onFileRemove evaluates if path was removed or renamed and trigger corresponding event
func (p *fsPoller) onFileRemove(path string, created map[string]*fs.FileInfo, oldFi *fs.FileInfo, renamed map[string]*fs.FileInfo) {
	for newPath, newFi := range created {
		if sameFile(*oldFi, *newFi) {
			err := p.SendEvent(Event{Op: Rename, Name: path, NewPath: newPath})
			if err != nil {
				p.errors <- err
			}
			renamed[newPath] = newFi
			return
		}
	}
	err := p.SendEvent(Event{Op: Remove, Name: path})
	if err != nil {
		p.errors <- err
	}
}

func (p *fsPoller) SendEvent(e Event) error {
	select {
	case p.events <- e:
	case <-p.done:
		return fmt.Errorf("watcher is closed")
	}
	return nil
}

// separate splits files into 2 categories: added and updated
func (p *fsPoller) separate(newFiles, added, updated map[string]*fs.FileInfo) {
	for file, info := range newFiles {
		_, exists := p.files[file]
		if !exists {
			added[file] = info
			continue
		}
		updated[file] = info
	}
}

// WatchedList return list of watched files and folders
func (p *fsPoller) WatchedList() map[string]*fs.FileInfo {
	p.mu.Lock()
	defer p.mu.Unlock()

	files := make(map[string]*fs.FileInfo)
	for k, v := range p.files {
		files[k] = v
	}

	return files
}

// watch watches item for changes until done is closed.
func (p *fsPoller) Start(interval time.Duration) error {
	if interval < MIN_INTERVAL {
		interval = MIN_INTERVAL
	}

	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return errors.New("watcher is already running")
	}
	p.running = true
	p.mu.Unlock()

	for {
		time.Sleep(interval)

		if p.closed {
			return nil
		}

		p.scanForChanges()
		select {
		case p.scanDone <- struct{}{}:
		case <-p.done:
			return nil
		}
	}
}

func (p *fsPoller) Remove(name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.files, name)
	return nil
}

func (p *fsPoller) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.running {
		return nil
	}
	p.running = false

	if p.closed {
		return nil
	}
	close(p.done)
	p.closed = true
	return nil
}

func (p *fsPoller) Errors() <-chan error {
	return p.errors
}

func (p *fsPoller) Events() <-chan Event {
	return p.events
}

func (p *fsPoller) ScanComplete() <-chan struct{} {
	return p.scanDone
}
