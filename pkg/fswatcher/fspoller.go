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

// fsPoller is polling implementing of FileWatcher interface
type fsPoller struct {
	// watched files and dirs
	watches map[string]struct{}
	// files and dirs inside watched paths
	files  map[string]*fs.FileInfo
	events chan Event
	errors chan error
	done   chan struct{}
	fsys   fs.FS
	// path to the root directory
	root    string
	running bool

	mu     *sync.Mutex
	closed bool
}

// Add adds given name into the list of the watched paths
// and saves FileInfo of nested files
func (p *fsPoller) Add(name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return errors.New("poller is closed")
	}

	relativePath, err := filepath.Rel(p.root, name)
	if err != nil {
		return err
	}

	list, err := p.listDirFiles(relativePath)
	if err != nil /* && errors.Is(err, fs.ErrNotExist) */ {
		return err
	}

	for fname, fi := range list {
		p.files[fname] = fi
	}

	p.watches[relativePath] = struct{}{}

	return nil
}

// listDirFiles returns list of the files if name is a directory,
// if name isn't a directory then just returns it's FileInfo.
func (p *fsPoller) listDirFiles(name string) (map[string]*fs.FileInfo, error) {
	files := map[string]*fs.FileInfo{}

	fInfo, err := fs.Stat(p.fsys, name)
	if err != nil {
		return nil, err
	}
	files[name] = &fInfo

	if !fInfo.IsDir() {
		return files, nil
	}

	dirEntires, err := fs.ReadDir(p.fsys, name)
	if err != nil {
		return nil, err
	}

	for _, de := range dirEntires {
		path := filepath.Join(name, de.Name())
		stat, err := de.Info()
		if err == nil {
			files[path] = &stat
		}
	}

	return files, nil
}

// scanForChanges checks folders for changes
func (p *fsPoller) scanForChanges() {
	p.mu.Lock()
	defer p.mu.Unlock()

	addedFiles := map[string]*fs.FileInfo{}
	updated := map[string]*fs.FileInfo{}

	for path := range p.watches {
		files, err := p.listDirFiles(path)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				p.onWatchedPathRemoved(path)
				continue
			}
			p.errors <- err
			continue
		}
		p.separate(files, addedFiles, updated)
	}

	for path, oldInfo := range p.files {
		newInfo, existed := updated[path]
		if existed {
			p.onFileWrite(path, oldInfo, newInfo)
			continue
		}
		p.onFileRemove(path, addedFiles, oldInfo)
	}

	for name, info := range addedFiles {
		p.files[name] = info
		p.sendEvent(Event{Op: Create, Name: name})
	}
}

// onWatchedPathRemoved removes path from watched paths and of its files/subfolder
func (p *fsPoller) onWatchedPathRemoved(path string) {
	delete(p.watches, path)
	info := p.files[path]
	if info != nil && !(*info).IsDir() {
		return
	}
	for file := range p.files {
		if filepath.Dir(file) == path {
			delete(p.files, file)
		}
	}
}

// onFileWrite checks if file with given path was changed, if positive triggers Write event
func (p *fsPoller) onFileWrite(path string, oldFi, newFi *fs.FileInfo) {
	if (*oldFi).ModTime() != (*newFi).ModTime() {
		p.files[path] = newFi
		p.sendEvent(Event{Op: Write, Name: path})
	}
}

// onFileRemove evaluates if path was removed or renamed and trigger corresponding event
func (p *fsPoller) onFileRemove(path string, created map[string]*fs.FileInfo, oldFi *fs.FileInfo) {
	delete(p.files, path)
	for newPath, newFi := range created {
		if sameFile(*oldFi, *newFi) {
			p.files[newPath] = newFi
			p.sendEvent(Event{Op: Rename, Name: path, NewPath: newPath})
			delete(created, newPath)
			return
		}
	}
	p.sendEvent(Event{Op: Remove, Name: path})
}

func (p *fsPoller) sendEvent(e Event) error {
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
