package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ErrInteractiveDisabled is returned when interactive prompts are disabled via STACKIT_TEST_NO_INTERACTIVE
var ErrInteractiveDisabled = fmt.Errorf("interactive prompts are disabled (STACKIT_TEST_NO_INTERACTIVE is set)")

// checkInteractiveAllowed returns an error if interactive mode is disabled for testing
func checkInteractiveAllowed() error {
	if os.Getenv("STACKIT_TEST_NO_INTERACTIVE") != "" {
		return ErrInteractiveDisabled
	}
	return nil
}

// textInputModel is a simple text input prompt model
type textInputModel struct {
	textInput textinput.Model
	prompt    string
	done      bool
	err       error
}

func (m textInputModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m textInputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.Type {
		case tea.KeyEnter:
			m.done = true
			return m, tea.Quit
		case tea.KeyCtrlC, tea.KeyEsc:
			m.err = fmt.Errorf("canceled")
			m.done = true
			return m, tea.Quit
		}
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m textInputModel) View() string {
	if m.done {
		return ""
	}
	style := lipgloss.NewStyle().Margin(1, 0)
	return style.Render(fmt.Sprintf("%s\n%s\n\n(Press Enter to submit, Ctrl+C to cancel)", m.prompt, m.textInput.View()))
}

// confirmModel is a simple yes/no confirmation prompt model
type confirmModel struct {
	prompt string
	choice bool
	done   bool
	err    error
}

func (m confirmModel) Init() tea.Cmd {
	return nil
}

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.Type {
		case tea.KeyEnter:
			m.done = true
			return m, tea.Quit
		case tea.KeyCtrlC, tea.KeyEsc:
			m.err = fmt.Errorf("canceled")
			m.done = true
			return m, tea.Quit
		case tea.KeyRunes:
			switch strings.ToLower(string(msg.Runes)) {
			case "y", "yes":
				m.choice = true
				m.done = true
				return m, tea.Quit
			case "n", "no":
				m.choice = false
				m.done = true
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m confirmModel) View() string {
	if m.done {
		return ""
	}
	style := lipgloss.NewStyle().Margin(1, 0)
	yesNo := "[y/N]"
	if m.choice {
		yesNo = "[Y/n]"
	}
	return style.Render(fmt.Sprintf("%s %s\n\n(Press y/yes or n/no, Enter to confirm, Ctrl+C to cancel)", m.prompt, yesNo))
}

// PromptTextInput prompts the user for text input
func PromptTextInput(prompt, defaultValue string) (string, error) {
	if err := checkInteractiveAllowed(); err != nil {
		return "", err
	}

	ti := textinput.New()
	ti.Placeholder = ""
	ti.SetValue(defaultValue)
	ti.Focus()
	ti.CharLimit = 500
	ti.Width = 80

	m := textInputModel{
		textInput: ti,
		prompt:    prompt,
	}

	p := tea.NewProgram(m, tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout))
	model, err := p.Run()
	if err != nil {
		return "", err
	}

	if finalModel, ok := model.(textInputModel); ok {
		if finalModel.err != nil {
			return "", finalModel.err
		}
		return finalModel.textInput.Value(), nil
	}

	return "", fmt.Errorf("unexpected model type")
}

// PromptConfirm prompts the user for yes/no confirmation
func PromptConfirm(prompt string, defaultValue bool) (bool, error) {
	if err := checkInteractiveAllowed(); err != nil {
		return false, err
	}

	m := confirmModel{
		prompt: prompt,
		choice: defaultValue,
	}

	p := tea.NewProgram(m, tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout))
	model, err := p.Run()
	if err != nil {
		return false, err
	}

	if finalModel, ok := model.(confirmModel); ok {
		if finalModel.err != nil {
			return false, finalModel.err
		}
		return finalModel.choice, nil
	}

	return false, fmt.Errorf("unexpected model type")
}

// SelectOption represents an option in a selection prompt
type SelectOption struct {
	Label string // What to show
	Value string // Value to return
}

// selectModel is a selection prompt model with arrow key navigation
type selectModel struct {
	options  []SelectOption
	cursor   int
	selected string
	done     bool
	err      error
	title    string
}

func (m selectModel) Init() tea.Cmd {
	return nil
}

func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.Type {
		case tea.KeyEnter:
			if len(m.options) > 0 && m.cursor >= 0 && m.cursor < len(m.options) {
				m.selected = m.options[m.cursor].Value
				m.done = true
				return m, tea.Quit
			}
		case tea.KeyCtrlC, tea.KeyEsc:
			m.err = fmt.Errorf("canceled")
			m.done = true
			return m, tea.Quit
		case tea.KeyUp, tea.KeyShiftTab:
			if m.cursor > 0 {
				m.cursor--
			} else {
				m.cursor = len(m.options) - 1
			}
			return m, nil
		case tea.KeyDown, tea.KeyTab:
			if m.cursor < len(m.options)-1 {
				m.cursor++
			} else {
				m.cursor = 0
			}
			return m, nil
		}
	}
	return m, nil
}

func (m selectModel) View() string {
	if m.done {
		return ""
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s\n\n", m.title))

	for i, opt := range m.options {
		if i == m.cursor {
			b.WriteString(fmt.Sprintf("  → %s\n", opt.Label))
		} else {
			b.WriteString(fmt.Sprintf("    %s\n", opt.Label))
		}
	}

	b.WriteString("\n(↑/↓ to select, Enter to confirm, Ctrl+C to cancel)")

	style := lipgloss.NewStyle().Margin(1, 0)
	return style.Render(b.String())
}

// PromptSelect prompts the user to select from a list of options
func PromptSelect(title string, options []SelectOption, defaultIndex int) (string, error) {
	if err := checkInteractiveAllowed(); err != nil {
		return "", err
	}

	if len(options) == 0 {
		return "", fmt.Errorf("no options provided")
	}

	cursor := defaultIndex
	if cursor < 0 || cursor >= len(options) {
		cursor = 0
	}

	m := selectModel{
		options: options,
		cursor:  cursor,
		title:   title,
	}

	p := tea.NewProgram(m, tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout))
	model, err := p.Run()
	if err != nil {
		return "", err
	}

	if finalModel, ok := model.(selectModel); ok {
		if finalModel.err != nil {
			return "", finalModel.err
		}
		return finalModel.selected, nil
	}

	return "", fmt.Errorf("unexpected model type")
}

// branchSelectModel is a branch selection prompt model with filtering
type branchSelectModel struct {
	choices  []BranchChoice
	filtered []BranchChoice
	filter   string
	cursor   int
	selected string
	done     bool
	err      error
	message  string
}

// BranchChoice represents a branch option in a selection prompt
type BranchChoice struct {
	Display string // What to show (may include tree visualization)
	Value   string // Actual branch name
}

func (m branchSelectModel) Init() tea.Cmd {
	return nil
}

func (m branchSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.Type {
		case tea.KeyEnter:
			if len(m.filtered) > 0 && m.cursor >= 0 && m.cursor < len(m.filtered) {
				m.selected = m.filtered[m.cursor].Value
				m.done = true
				return m, tea.Quit
			}
		case tea.KeyCtrlC, tea.KeyEsc:
			m.err = fmt.Errorf("canceled")
			m.done = true
			return m, tea.Quit
		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			} else {
				m.cursor = len(m.filtered) - 1
			}
			return m, nil
		case tea.KeyDown:
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			} else {
				m.cursor = 0
			}
			return m, nil
		case tea.KeyBackspace:
			if len(m.filter) > 0 {
				m.filter = m.filter[:len(m.filter)-1]
				m.updateFiltered()
				if m.cursor >= len(m.filtered) {
					m.cursor = len(m.filtered) - 1
				}
				if m.cursor < 0 {
					m.cursor = 0
				}
			}
			return m, nil
		case tea.KeyRunes:
			m.filter += string(msg.Runes)
			m.updateFiltered()
			if m.cursor >= len(m.filtered) {
				m.cursor = len(m.filtered) - 1
			}
			if m.cursor < 0 {
				m.cursor = 0
			}
			return m, nil
		}
	}
	return m, nil
}

func (m *branchSelectModel) updateFiltered() {
	if m.filter == "" {
		m.filtered = m.choices
		return
	}

	filterLower := strings.ToLower(m.filter)
	m.filtered = []BranchChoice{}
	for _, choice := range m.choices {
		if strings.Contains(strings.ToLower(choice.Display), filterLower) ||
			strings.Contains(strings.ToLower(choice.Value), filterLower) {
			m.filtered = append(m.filtered, choice)
		}
	}
}

func (m branchSelectModel) View() string {
	if m.done {
		return ""
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s\n", m.message))

	if m.filter != "" {
		b.WriteString(fmt.Sprintf("Filter: %s\n\n", m.filter))
	}

	if len(m.filtered) == 0 {
		b.WriteString("No branches match the filter.\n")
	} else {
		for i, choice := range m.filtered {
			cursor := " "
			if i == m.cursor {
				cursor = ">"
			}
			b.WriteString(fmt.Sprintf("%s %s\n", cursor, choice.Display))
		}
	}

	b.WriteString("\n(Press Enter to select, Ctrl+C to cancel, type to filter)")

	style := lipgloss.NewStyle().Margin(1, 0)
	return style.Render(b.String())
}

// PromptBranchSelection prompts the user to select a branch
func PromptBranchSelection(message string, choices []BranchChoice, initialIndex int) (string, error) {
	if err := checkInteractiveAllowed(); err != nil {
		return "", err
	}

	m := branchSelectModel{
		choices: choices,
		filter:  "",
		cursor:  initialIndex,
		message: message,
	}
	m.updateFiltered()

	// Adjust cursor to initial index in filtered list
	if initialIndex >= 0 && initialIndex < len(choices) {
		initialChoice := choices[initialIndex]
		for i, filtered := range m.filtered {
			if filtered.Value == initialChoice.Value {
				m.cursor = i
				break
			}
		}
	}

	if m.cursor < 0 || m.cursor >= len(m.filtered) {
		if len(m.filtered) > 0 {
			m.cursor = 0
		}
	}

	p := tea.NewProgram(m, tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout))
	model, err := p.Run()
	if err != nil {
		return "", err
	}

	if finalModel, ok := model.(branchSelectModel); ok {
		if finalModel.err != nil {
			return "", finalModel.err
		}
		return finalModel.selected, nil
	}

	return "", fmt.Errorf("unexpected model type")
}
