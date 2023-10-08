package teaprogram

import tea "github.com/charmbracelet/bubbletea"

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

type logMsg string

func waitForLogs(logChan chan string) tea.Cmd {
	return func() tea.Msg {
		return logMsg(<-logChan)
	}
}
