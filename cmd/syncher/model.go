package teaprogram

import (
	"fmt"
	imagesync "imagesync/pkg/sync"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gookit/color"
)

type movesMsg map[string]string

type model struct {
	pollinterval time.Duration
	syncer       *imagesync.ImageSync
	root         string
	quitting     bool
	watching     bool
	moves        chan movesMsg
}

func listenForMoves(m model) tea.Cmd {
	return func() tea.Msg {
		m.syncer.WatchEvents(func(moves map[string]string) {
			movesCopy := make(map[string]string)
			for k, v := range moves {
				movesCopy[k] = v
			}
			m.moves <- movesMsg(movesCopy)
		})
		return nil
	}
}

func waitForMoves(mv chan movesMsg) tea.Cmd {
	return func() tea.Msg {
		return movesMsg(<-mv)
	}
}

// Init optionally returns an initial command we should run.
func (m model) Init() tea.Cmd {
	m.syncer.ProcessFiles()
	cmds := []tea.Cmd{
		listenForMoves(m),
		waitForMoves(m.moves),
	}
	if m.watching {
		cmds = append(cmds, watch(m))
		return tea.Batch(cmds...)
	}
	return tea.Batch(cmds...)
}

func watch(m model) tea.Cmd {
	return func() tea.Msg {
		go m.syncer.StartFileWatcher(time.Millisecond * 500)
		return nil
	}
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
		case "enter":
			if m.watching {
				return m, nil
			}
			m.syncer.Scan()
			return m, nil
		}
	case movesMsg:
		if m.watching {
			m.syncer.Sync(msg)
		} else {
			// TODO:
			fmt.Println("TODO: should sync in manual mode")
		}
		return m, waitForMoves(m.moves)
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

func NewProgram(root string, interval time.Duration) *tea.Program {
	syncer := imagesync.New(os.DirFS(root), root)
	watching := interval > 0
	return tea.NewProgram(model{
		root:         root,
		syncer:       syncer,
		watching:     watching,
		pollinterval: interval,
		moves:        make(chan movesMsg)})
}
