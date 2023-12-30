package teaprogram

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/flytaly/imagesync/pkg/log"
	imagesync "github.com/flytaly/imagesync/pkg/sync"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gookit/color"
)

type Status int

const (
	Initial Status = iota
	Watching
	Waiting
	ShouldConfirm
	Quitting
)

var (
	logRosShow    = 6
	logRowsTotal  = 30
	logWidth      = 40
	spinnerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	dotStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Margin(0, 0, 0, 0)
	logTitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Bold(true).Width(logWidth).Align(lipgloss.Center).Margin(0, 0, 0, 0)
	logTextStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	logErrorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	appStyle      = lipgloss.NewStyle().Margin(1, 1, 0, 1)
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

	logCh   chan log.Record
	logs    []log.Record
	showLog bool

	duration time.Duration
}

// Init optionally returns an initial command we should run.
func (m model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		processFiles(m),
		listenForMoves(m),
		waitForMoves(m.movesChan),
		waitForLogs(m.logCh),
		m.spinner.Tick,
	}

	return tea.Batch(cmds...)
}

type fileProcessed struct {
	d time.Duration
}

func processFiles(m model) tea.Cmd {
	return func() tea.Msg {
		d := m.syncer.ProcessFiles()
		return fileProcessed{d}
	}
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
		case key.Matches(msg, m.keys.Log):
			m.showLog = !m.showLog
			return m, nil
		}
	case fileProcessed:
		m.duration = msg.d
		if m.pollinterval > 0 {
			m.status = Watching
			return m, tea.Batch(watch(m), m.spinner.Tick)
		}
		m.status = Waiting
		return m, nil
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
	case log.Record:
		if len(m.logs) >= logRowsTotal {
			m.logs = m.logs[1:]
		}
		m.logs = append(m.logs, msg)
		return m, waitForLogs(m.logCh)
	case spinner.TickMsg:
		if m.status != Watching && m.status != Initial {
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

	if m.status != Initial {
		result += m.renderStats()
	}

	switch m.status {
	case Initial:
		result += fmt.Sprintf("\n%s Scanning directory...", m.spinner.View())
		return result
	case Watching:
		result += fmt.Sprintf("\n\n%s Watch for changes", m.spinner.View())
	case ShouldConfirm:
		result += "\nMoves:\n" + printMoves(m.moves, 6, 80)
		result += fmt.Sprintf("\n\n%s Press '%s' to update links or %s to skip", color.Green.Sprint("➜"), color.Green.Sprint("y"), color.Green.Sprint("n"))
	case Waiting:
		result += fmt.Sprintf("\n\n%s Press '%s' to check the path for changes", color.Green.Sprint("➜"), color.Green.Sprint("Enter"))
	}

	helpView := m.help.View(m.keys)

	result += "\n\n" + helpView

	if m.showLog {
		result += "\n\n" + m.renderLog()
	}

	return appStyle.Render(result)
}

func (m model) renderStats() string {
	result := logTextStyle.Render(fmt.Sprintf("\n%d source files. %d linked images.",
		m.syncer.SourcesNum(),
		m.syncer.RefsNum(),
	))

	if m.duration > time.Second {
		return result + logErrorStyle.Render(fmt.Sprintf(" [%.1f seconds]", m.duration.Seconds()))
	}

	return result + logTextStyle.Render(fmt.Sprintf("[%d ms]", m.duration.Milliseconds()))
}

func (m model) renderLog() string {
	result := logTitleStyle.Render("Log") + "\n"
	offset := max(len(m.logs)-logRosShow, 0)
	for i := len(m.logs) - 1; i >= offset; i-- {
		r := fmt.Sprintf("[%s] %s", m.logs[i].Ts.Format("15:04:05"), m.logs[i].Msg)
		switch m.logs[i].Level {
		case log.Info:
			result += logTextStyle.Render("✓ " + r)
		case log.Error:
			result += logErrorStyle.Render("✕ " + r)
		case log.Warning:
			result += logTextStyle.Render("⚠ " + r)
		}
		result += "\n"
	}
	for i := 0; i < logRosShow-offset; i++ {
		result += fmt.Sprintf("%s\n", dotStyle.Render(strings.Repeat(".", logWidth)))
	}
	return result
}

type ProgramCfg struct {
	Interval    time.Duration
	LogPath     string
	Root        string
	MaxFileSize int64
}

func NewProgram(cfg ProgramCfg) *tea.Program {
	logChannel := make(chan log.Record, logRowsTotal)
	syncer := imagesync.New(
		os.DirFS(cfg.Root), cfg.Root, log.New(cfg.LogPath, logChannel),
		func(s *imagesync.ImageSync) {
			if cfg.MaxFileSize > 0 {
				s.MaxFileSize = cfg.MaxFileSize
			}
		},
	)

	helpModel := help.New()

	return tea.NewProgram(model{
		keys:         keys,
		help:         helpModel,
		root:         cfg.Root,
		syncer:       syncer,
		pollinterval: cfg.Interval,
		status:       Initial,
		movesChan:    make(chan movesMsg),
		spinner:      newSpinner(),
		logCh:        logChannel,
		logs:         []log.Record{},
		showLog:      true,
	}) //, tea.WithAltScreen())
}

func newSpinner() spinner.Model {
	s := spinner.New()
	s.Spinner = spinner.Line
	s.Style = spinnerStyle
	return s
}
