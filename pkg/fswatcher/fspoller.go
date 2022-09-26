package fswatcher

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
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
	files  map[string]os.FileInfo
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
// if name isn't a directory then just return it.
func (p *fsPoller) listDirFiles(name string) (map[string]os.FileInfo, error) {
	files := map[string]os.FileInfo{}

	fInfo, err := fs.Stat(p.fsys, name)
	if err != nil {
		return nil, err
	}
	files[name] = fInfo

	if !fInfo.IsDir() {
		return files, nil
	}

	dirEntires, err := fs.ReadDir(p.fsys, name)
	if err != nil {
		return nil, err
	}

	for _, de := range dirEntires {
		path := filepath.Join(name, de.Name())
		files[path] = fInfo
	}

	return files, nil
}

// checkChanges checks folders for changes
func (p *fsPoller) checkChanges() {
	// files := p.WatchedList()
	// TODO:
	fmt.Println("check for changes", time.Now())
}

// WatchedList return list of watched files and folders
func (p *fsPoller) WatchedList() map[string]os.FileInfo {
	p.mu.Lock()
	defer p.mu.Unlock()

	files := make(map[string]os.FileInfo)
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

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			p.checkChanges()

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
