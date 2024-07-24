package teaprogram

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/flytaly/linksyncer/pkg/log"
)

type movesMsg map[string]string

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

func waitForLogs(logChan chan log.Record) tea.Cmd {
	return func() tea.Msg {
		return log.Record(<-logChan)
	}
}
