package teaprogram

import (
	"fmt"
	"imagesync/pkg/log"
	imagesync "imagesync/pkg/sync"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gookit/color"
)

type Status int

const (
	Watching Status = iota
	Waiting
	ShouldConfirm
	Quitting
)

type model struct {
	keys         keyMap
	help         help.Model
	pollinterval time.Duration
	syncer       *imagesync.ImageSync
	root         string
	status       Status
	movesChan    chan movesMsg
	moves        map[string]string

	spinner spinner.Model
	logCh   chan string
	logs    []string
}

// Init optionally returns an initial command we should run.
func (m model) Init() tea.Cmd {
	m.syncer.ProcessFiles()

	cmds := []tea.Cmd{
		listenForMoves(m),
		waitForMoves(m.movesChan),
		waitForLogs(m.logCh),
	}
	if m.status == Watching {
		cmds = append(cmds, watch(m), m.spinner.Tick)
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
		switch {
		case key.Matches(msg, m.keys.Quit):
			m.status = Quitting
			m.syncer.Close()
			return m, tea.Quit
		case key.Matches(msg, m.keys.Watch):
			if m.status == Watching {
				m.syncer.StopFileWatcher()
				m.status = Waiting
				return m, nil
			}
			m.syncer.ProcessFiles()
			m.status = Watching
			return m, tea.Batch(watch(m), m.spinner.Tick)
		case key.Matches(msg, m.keys.Confirm):
			if m.status == Watching {
				return m, nil
			}
			if len(m.moves) != 0 {
				m.syncer.Sync(m.moves)
			}
			m.status = Waiting
			m.syncer.Scan()
			return m, nil
		case key.Matches(msg, m.keys.Cancel):
			if m.status == ShouldConfirm {
				m.status = Waiting
			}
			m.syncer.Scan()
			return m, nil
		}
	case tea.WindowSizeMsg:
		// If we set a width on the help menu it can gracefully truncate
		// its view as needed.
		m.help.Width = msg.Width
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
	case logMsg:
		if len(m.logs) > 20 {
			m.logs = m.logs[1:]
		}
		m.logs = append(m.logs, string(msg))
		return m, waitForLogs(m.logCh)
	case spinner.TickMsg:
		if m.status != Watching {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

// View returns a string based on data in the model. That string which will be
// rendered to the terminal.
func (m model) View() string {
	result := fmt.Sprintf("Path %s", color.Cyan.Sprint(m.root))
	switch m.status {
	case Watching:
		result += fmt.Sprintf("\n%s Watch for changes", m.spinner.View())
	case ShouldConfirm:
		result += "\nMoves:\n" + printMoves(m.moves, 6, 80)
		result += fmt.Sprintf("\n%s Press '%s' to update links or %s to skip", color.Green.Sprint("➜"), color.Green.Sprint("y"), color.Green.Sprint("n"))
	case Waiting:
		result += fmt.Sprintf("\n%s Press '%s' to check the path for changes", color.Green.Sprint("➜"), color.Green.Sprint("Enter"))
	}

	helpView := m.help.View(m.keys)

	result += "\n\n" + helpView

	if len(m.logs) > 0 {
		result += "\n\nLogs:\n"
		offset := max(len(m.logs)-6, 0)
		for i := len(m.logs) - 1; i >= offset; i-- {
			result += fmt.Sprintf("%s\n", m.logs[i])
		}
	}
	return result
}

func NewProgram(root string, interval time.Duration) *tea.Program {
	logChannel := make(chan string, 10)
	syncer := imagesync.New(os.DirFS(root), root, log.New("info.log", logChannel))

	status := Waiting
	if interval > 0 {
		status = Watching
	}

	helpModel := help.New()

	return tea.NewProgram(model{
		keys:         keys,
		help:         helpModel,
		root:         root,
		syncer:       syncer,
		pollinterval: interval,
		status:       status,
		movesChan:    make(chan movesMsg),
		spinner:      newSpinner(),
		logCh:        logChannel,
		logs:         []string{},
	}) //, tea.WithAltScreen())
}

func newSpinner() spinner.Model {
	s := spinner.New()
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	s.Spinner = spinner.Line
	return s
}
