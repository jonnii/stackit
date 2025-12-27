package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/tui/components/tree"
	"stackit.dev/stackit/internal/tui/style"
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
	styleObj := lipgloss.NewStyle().Margin(1, 0)
	return styleObj.Render(fmt.Sprintf("%s\n%s\n\n(Press Enter to submit, Ctrl+C to cancel)", m.prompt, m.textInput.View()))
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
	styleObj := lipgloss.NewStyle().Margin(1, 0)
	yesNo := "[y/N]"
	if m.choice {
		yesNo = "[Y/n]"
	}
	return styleObj.Render(fmt.Sprintf("%s %s\n\n(Press y/yes or n/no, Enter to confirm, Ctrl+C to cancel)", m.prompt, yesNo))
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

// SelectModel is a selection prompt model with arrow key navigation
type SelectModel struct {
	Options  []SelectOption
	Cursor   int
	Selected string
	Done     bool
	Err      error
	Title    string
}

// Init initializes the bubbletea model
func (m SelectModel) Init() tea.Cmd {
	return nil
}

// Update handles message updates for the bubbletea model
func (m SelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.Type {
		case tea.KeyEnter:
			if len(m.Options) > 0 && m.Cursor >= 0 && m.Cursor < len(m.Options) {
				m.Selected = m.Options[m.Cursor].Value
				m.Done = true
				return m, tea.Quit
			}
		case tea.KeyCtrlC, tea.KeyEsc:
			m.Err = fmt.Errorf("canceled")
			m.Done = true
			return m, tea.Quit
		case tea.KeyUp, tea.KeyShiftTab:
			if m.Cursor > 0 {
				m.Cursor--
			} else {
				m.Cursor = len(m.Options) - 1
			}
			return m, nil
		case tea.KeyDown, tea.KeyTab:
			if m.Cursor < len(m.Options)-1 {
				m.Cursor++
			} else {
				m.Cursor = 0
			}
			return m, nil
		}
	}
	return m, nil
}

// View renders the TUI
func (m SelectModel) View() string {
	if m.Done {
		return ""
	}

	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render(m.Title))
	b.WriteString("\n\n")

	for i, opt := range m.Options {
		if i == m.Cursor {
			b.WriteString(fmt.Sprintf("  → %s\n", lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render(opt.Label)))
		} else {
			b.WriteString(fmt.Sprintf("    %s\n", opt.Label))
		}
	}

	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("\n(↑/↓ to select, Enter to confirm, Ctrl+C to cancel)"))

	styleObj := lipgloss.NewStyle().Margin(1, 0)
	return styleObj.Render(b.String())
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

	m := SelectModel{
		Options: options,
		Cursor:  cursor,
		Title:   title,
	}

	p := tea.NewProgram(m, tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout))
	model, err := p.Run()
	if err != nil {
		return "", err
	}

	if finalModel, ok := model.(SelectModel); ok {
		if finalModel.Err != nil {
			return "", finalModel.Err
		}
		return finalModel.Selected, nil
	}

	return "", fmt.Errorf("unexpected model type")
}

// BranchSelectModel is a branch selection prompt model with filtering
type BranchSelectModel struct {
	Choices  []BranchChoice
	Filtered []BranchChoice
	Filter   string
	Cursor   int
	Selected string
	Done     bool
	Err      error
	Message  string
}

// BranchChoice represents a branch option in a selection prompt
type BranchChoice struct {
	Display string // What to show (may include tree visualization)
	Value   string // Actual branch name
}

// Init initializes the bubbletea model
func (m BranchSelectModel) Init() tea.Cmd {
	return nil
}

// Update handles message updates for the bubbletea model
func (m BranchSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.Type {
		case tea.KeyEnter:
			if len(m.Filtered) > 0 && m.Cursor >= 0 && m.Cursor < len(m.Filtered) {
				m.Selected = m.Filtered[m.Cursor].Value
				m.Done = true
				return m, tea.Quit
			}
		case tea.KeyCtrlC, tea.KeyEsc:
			m.Err = fmt.Errorf("canceled")
			m.Done = true
			return m, tea.Quit
		case tea.KeyUp:
			if m.Cursor > 0 {
				m.Cursor--
			} else {
				m.Cursor = len(m.Filtered) - 1
			}
			return m, nil
		case tea.KeyDown:
			if m.Cursor < len(m.Filtered)-1 {
				m.Cursor++
			} else {
				m.Cursor = 0
			}
			return m, nil
		case tea.KeyBackspace:
			if len(m.Filter) > 0 {
				m.Filter = m.Filter[:len(m.Filter)-1]
				m.updateFiltered()
				if m.Cursor >= len(m.Filtered) {
					m.Cursor = len(m.Filtered) - 1
				}
				if m.Cursor < 0 {
					m.Cursor = 0
				}
			}
			return m, nil
		case tea.KeyRunes:
			m.Filter += string(msg.Runes)
			m.updateFiltered()
			if m.Cursor >= len(m.Filtered) {
				m.Cursor = len(m.Filtered) - 1
			}
			if m.Cursor < 0 {
				m.Cursor = 0
			}
			return m, nil
		}
	}
	return m, nil
}

func (m *BranchSelectModel) updateFiltered() {
	if m.Filter == "" {
		m.Filtered = m.Choices
		return
	}

	filterLower := strings.ToLower(m.Filter)
	m.Filtered = []BranchChoice{}
	for _, choice := range m.Choices {
		if strings.Contains(strings.ToLower(choice.Display), filterLower) ||
			strings.Contains(strings.ToLower(choice.Value), filterLower) {
			m.Filtered = append(m.Filtered, choice)
		}
	}
}

// View renders the TUI
func (m BranchSelectModel) View() string {
	if m.Done {
		return ""
	}

	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render(m.Message))
	b.WriteString("\n")

	if m.Filter != "" {
		b.WriteString(fmt.Sprintf("Filter: %s\n\n", lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render(m.Filter)))
	} else {
		b.WriteString("\n")
	}

	if len(m.Filtered) == 0 {
		b.WriteString("No branches match the filter.\n")
	} else {
		for i, choice := range m.Filtered {
			cursor := " "
			if i == m.Cursor {
				cursor = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render(">")
			}
			b.WriteString(fmt.Sprintf("%s %s\n", cursor, choice.Display))
		}
	}

	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("\n(Press Enter to select, Ctrl+C to cancel, type to filter)"))

	styleObj := lipgloss.NewStyle().Margin(1, 0)
	return styleObj.Render(b.String())
}

// PromptBranchSelection prompts the user to select a branch
func PromptBranchSelection(message string, choices []BranchChoice, initialIndex int) (string, error) {
	if err := checkInteractiveAllowed(); err != nil {
		return "", err
	}

	m := BranchSelectModel{
		Choices: choices,
		Filter:  "",
		Cursor:  initialIndex,
		Message: message,
	}
	m.updateFiltered()

	// Adjust cursor to initial index in filtered list
	if initialIndex >= 0 && initialIndex < len(choices) {
		initialChoice := choices[initialIndex]
		for i, filtered := range m.Filtered {
			if filtered.Value == initialChoice.Value {
				m.Cursor = i
				break
			}
		}
	}

	if m.Cursor < 0 || m.Cursor >= len(m.Filtered) {
		if len(m.Filtered) > 0 {
			m.Cursor = 0
		}
	}

	p := tea.NewProgram(m, tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout))
	model, err := p.Run()
	if err != nil {
		return "", err
	}

	if finalModel, ok := model.(BranchSelectModel); ok {
		if finalModel.Err != nil {
			return "", finalModel.Err
		}
		return finalModel.Selected, nil
	}

	return "", fmt.Errorf("unexpected model type")
}

// PromptBranchCheckout shows an interactive branch selector for checkout.
// It takes a list of branches and the engine context, formats them using tree rendering,
// and presents them for selection.
func PromptBranchCheckout(branches []engine.Branch, eng engine.BranchReader) (string, error) {
	if len(branches) == 0 {
		return "", fmt.Errorf("no branches available to checkout")
	}

	// Create tree renderer
	currentBranch := eng.CurrentBranch()
	trunk := eng.Trunk()
	renderer := NewStackTreeRenderer(eng)

	// Add annotations for all branches
	annotations := make(map[string]tree.BranchAnnotation)
	for _, branch := range branches {
		scopeStr := eng.GetScopeInternal(branch.GetName())
		if !scopeStr.IsEmpty() {
			annotations[branch.GetName()] = tree.BranchAnnotation{
				Scope: scopeStr.String(),
			}
		}
	}
	renderer.SetAnnotations(annotations)

	// Calculate depth for each branch to create proper tree indentation
	branchDepth := make(map[string]int)
	branchDepth[trunk.GetName()] = 0

	// Build depth map by traversing from trunk
	var calculateDepth func(branchName string, depth int)
	calculateDepth = func(branchName string, depth int) {
		branch := eng.GetBranch(branchName)
		children := branch.GetChildren()
		for _, child := range children {
			branchDepth[child.GetName()] = depth + 1
			calculateDepth(child.GetName(), depth+1)
		}
	}
	calculateDepth(trunk.GetName(), 0)

	choices := make([]BranchChoice, 0, len(branches))
	initialIndex := -1

	for i, branch := range branches {
		isCurrent := currentBranch != nil && branch.GetName() == currentBranch.GetName()
		if isCurrent {
			initialIndex = i
		}

		// Get depth for indentation
		depth := branchDepth[branch.GetName()]

		// Create tree line with proper indentation
		indent := strings.Repeat("  ", depth)
		var symbol string
		if isCurrent {
			symbol = tree.CurrentBranchSymbol
		} else {
			symbol = tree.BranchSymbol
		}

		// Get colored branch name
		coloredBranchName := style.ColorBranchName(branch.GetName(), isCurrent)

		// Add annotation
		annotation := annotations[branch.GetName()]
		coloredBranchName += renderer.FormatAnnotationColored(annotation)

		// Add restack indicator if needed
		if !eng.IsBranchUpToDateInternal(branch.GetName()) {
			coloredBranchName += " " + style.ColorNeedsRestack("(needs restack)")
		}

		display := indent + symbol + " " + coloredBranchName

		choices = append(choices, BranchChoice{
			Display: display,
			Value:   branch.GetName(),
		})
	}

	// Set initial index if not found
	if initialIndex < 0 {
		initialIndex = len(choices) - 1
	}

	// Show interactive selector
	selected, err := PromptBranchSelection("Checkout a branch (arrow keys to navigate, type to filter)", choices, initialIndex)
	if err != nil {
		return "", err
	}

	return selected, nil
}
