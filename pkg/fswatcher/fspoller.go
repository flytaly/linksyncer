package fswatcher

import (
	"errors"
	"io/fs"
	"os"
	"sync"
)

// fsPoller is polling implementing of FileWatcher interface
type fsPoller struct {
	files   map[string]os.FileInfo
	watches map[string]struct{}
	events  chan Event
	errors  chan error
	close   chan struct{}
	fsys    fs.FS

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

	return nil
}
