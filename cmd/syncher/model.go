package teaprogram

import (
	"fmt"
	imagesync "imagesync/pkg/sync"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gookit/color"
)

type model struct {
	// choices  []string           // items on the to-do list
	// cursor   int                // which to-do list item our cursor is pointing at
	// selected map[int]struct{}   // which to-do items are selected
	pollinterval time.Duration
	syncer       *imagesync.ImageSync
	root         string
	quitting     bool
	watching     bool
}

// Init optionally returns an initial command we should run.
func (m model) Init() tea.Cmd {
	m.syncer.ProcessFiles()
	if m.watching {
		return watch(m)
	}
	return nil
}

func watch(m model) tea.Cmd {
	m.syncer.Watch(time.Millisecond * 500)
	return nil
}

// Update is called when messages are received. The idea is that you inspect the
// message and send back an updated model accordingly. You can also return
// a command, which is a function that performs I/O and returns a message.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.quitting = true
			m.syncer.Close()
			return m, tea.Quit
		case "w":
			if m.watching {
				m.syncer.StopFileWatcher()
				m.watching = false
				return m, nil
			}
			m.syncer.ProcessFiles()
			m.watching = true
			watch(m)
			return m, nil
		}
	case watchMsg:
		m.watching = true
		return m, nil
	}
	return m, nil
}

// View returns a string based on data in the model. That string which will be
// rendered to the terminal.
func (m model) View() string {
	if m.quitting {
		return ""
	}
	result := ""

	if m.watching {
		result = fmt.Sprintf(" %s  Watch path: %s", color.Green.Sprint("➜"), color.Cyan.Sprint(m.root))
	}

	return result
}

type watchMsg time.Duration

func NewProgram(root string, interval time.Duration) *tea.Program {
	syncer := imagesync.New(os.DirFS(root), root)
	watching := interval > 0
	return tea.NewProgram(model{root: root, syncer: syncer, watching: watching, pollinterval: interval})
}
