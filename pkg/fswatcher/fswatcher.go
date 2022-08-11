package fswatcher

import (
	"io/fs"
	"os"
	"sync"
)

// Event represents a single file system notification
type Event struct {
	Name string // Path to the file or directory
	Op   Op     // File operation that triggered the event.
}

// Op describes a type of event
type Op uint32

// Operations
const (
	Create Op = 1 << iota
	Write
	Remove
	Rename
	Move
	Chmod
)

func (op Op) String() string {
	switch op {
	case Create:
		return "CREATE"
	case Write:
		return "WRITE"
	case Remove:
		return "REMOVE"
	case Rename:
		return "RENAME"
	case Move:
		return "MOVE"
	case Chmod:
		return "CHMOD"
	}
	return "?"
}

// FsWatcher is fsnotify-like interface for implementing file watchers
type FsWatcher interface {
	Events() <-chan Event
	Errors() <-chan error
	Add(name string) error
	Remove(name string) error
	Close() error
}

// New creates a new Watcher.
func NewFsPoller(fsys fs.FS) *fsPoller {
	return &fsPoller{
		events: make(chan Event),
		errors: make(chan error),
		closed: false,
		close:  make(chan struct{}),
		fsys:   fsys,
		mu:     new(sync.Mutex),
		files:  make(map[string]os.FileInfo),
	}
}
