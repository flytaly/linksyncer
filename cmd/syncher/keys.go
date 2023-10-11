package teaprogram

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Confirm key.Binding
	Cancel  key.Binding
	Watch   key.Binding
	Quit    key.Binding
	Log     key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view. It's part
// of the key.Map interface.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Watch, k.Log, k.Quit}
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// key.Map interface.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{}
}

var keys = keyMap{
	Confirm: key.NewBinding(
		key.WithKeys("enter", "y"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc", "n"),
	),
	Watch: key.NewBinding(
		key.WithKeys("w"),
		key.WithHelp("w", "toggle watch mode"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Log: key.NewBinding(
		key.WithKeys("l"),
		key.WithHelp("l", "toggle log"),
	),
}
