package fswatcher

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sync"
	"time"
)

// fsPoller is polling implementing of FileWatcher interface
type fsPoller struct {
	files    map[string]os.FileInfo
	watches  map[string]struct{}
	events   chan Event
	errors   chan error
	done     chan struct{}
	fsys     fs.FS
	interval time.Duration

	mu     *sync.Mutex
	closed bool
}

func (p *fsPoller) Add(name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return errors.New("poller is closed")
	}

	fi, err := fs.Stat(p.fsys, name)
	if err != nil && errors.Is(err, fs.ErrNotExist) {
		return err
	}

	p.files[name] = fi

	go p.watch(name)

	return nil
}

func (p *fsPoller) check() {

}

// watch watches item for changes until done is closed.
func (p *fsPoller) watch(fname string) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			fmt.Println("tick")
		case <-p.done:
			return
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

	if p.closed {
		return nil
	}
	p.closed = true
	close(p.done)
	return nil
}

func (p *fsPoller) Errors() <-chan error {
	return p.errors
}

func (p *fsPoller) Events() <-chan Event {
	return p.events
}
