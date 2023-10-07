package teaprogram

import (
	"fmt"
	"imagesync/pkg/log"
	imagesync "imagesync/pkg/sync"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gookit/color"
)

type movesMsg map[string]string

type Status int

const (
	Watching Status = iota
	Waiting
	ShouldConfirm
	Quitting
)

type model struct {
	pollinterval time.Duration
	syncer       *imagesync.ImageSync
	root         string
	status       Status
	movesChan    chan movesMsg
	moves        map[string]string
}

func listenForMoves(m model) tea.Cmd {
	return func() tea.Msg {
		m.syncer.WatchEvents(func(moves map[string]string) {
			movesCopy := make(map[string]string)
			for k, v := range moves {
				movesCopy[k] = v
			}
			m.movesChan <- movesMsg(movesCopy)
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
		waitForMoves(m.movesChan),
	}
	if m.status == Watching {
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
		case "q", "ctrl+c":
			m.status = Quitting
			m.syncer.Close()
			return m, tea.Quit
		case "w":
			if m.status == Watching {
				m.syncer.StopFileWatcher()
				m.status = Waiting
				return m, nil
			}
			m.syncer.ProcessFiles()
			m.status = Watching
			watch(m)
			return m, nil
		case "enter", "y":
			if m.status == Watching {
				return m, nil
			}
			if len(m.moves) != 0 {
				m.syncer.Sync(m.moves)
			}
			m.status = Waiting
			m.syncer.Scan()
			return m, nil
		case "esc", "n":
			if m.status == ShouldConfirm {
				m.status = Waiting
			}
			m.syncer.Scan()
			return m, nil
		}
	case movesMsg:
		switch m.status {
		case Watching:
			m.syncer.Sync(msg)
		default:
			if len(msg) != 0 {
				m.status = ShouldConfirm
				m.moves = msg
			}
		}
		return m, waitForMoves(m.movesChan)
	}
	return m, nil
}

// View returns a string based on data in the model. That string which will be
// rendered to the terminal.
func (m model) View() string {
	result := fmt.Sprintf("Path %s", color.Cyan.Sprint(m.root))
	switch m.status {
	case Watching:
		result += fmt.Sprintf("\n%s Watch for changes", color.Green.Sprint("➜"))
	case ShouldConfirm:
		result += "\nMoves:\n" + printMoves(m.moves, 6, 70)
		result += fmt.Sprintf("\n%s Press %s to update links or %s to skip", color.Green.Sprint("➜"), color.Green.Sprint("y"), color.Green.Sprint("n"))
	case Waiting:
		result += fmt.Sprintf("\n%s Press %s to check the path for changes", color.Green.Sprint("➜"), color.Green.Sprint("Enter"))
	}
	return result
}

func NewProgram(root string, interval time.Duration) *tea.Program {
	syncer := imagesync.New(os.DirFS(root), root, log.New("info.log"))
	status := Waiting
	if interval > 0 {
		status = Watching
	}
	return tea.NewProgram(model{
		root:         root,
		syncer:       syncer,
		pollinterval: interval,
		status:       status,
		movesChan:    make(chan movesMsg)})
}
