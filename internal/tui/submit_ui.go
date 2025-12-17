package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SubmitUI defines the interface for the full submit workflow display
type SubmitUI interface {
	// ShowStack displays the stack to be submitted
	ShowStack(lines []string)

	// ShowRestack shows restack progress
	ShowRestackStart()
	ShowRestackComplete()

	// ShowPreparing shows the preparation phase
	ShowPreparing()

	// ShowBranchPlan shows what will happen to a branch
	ShowBranchPlan(branchName string, action string, isCurrent bool, skip bool, skipReason string)

	// ShowNoChanges indicates all PRs are up to date
	ShowNoChanges()

	// ShowDryRunComplete indicates dry run is complete
	ShowDryRunComplete()

	// StartSubmitting begins the submission phase
	StartSubmitting(items []SubmitItem)

	// UpdateSubmitItem updates status during submission
	UpdateSubmitItem(idx int, status string, url string, err error)

	// Complete finalizes and shows summary
	Complete()
}

// NewSubmitUI creates the appropriate UI based on TTY availability
func NewSubmitUI(splog *Splog) SubmitUI {
	if IsTTY() {
		return NewTTYSubmitUI(splog)
	}
	return NewSimpleSubmitUI(splog)
}

// ============================================================================
// SimpleSubmitUI - Non-bubbletea implementation for non-TTY environments
// ============================================================================

// SimpleSubmitUI implements SubmitUI with line-by-line output
type SimpleSubmitUI struct {
	splog     *Splog
	items     []SubmitItem
	completed int
	failed    int
}

// NewSimpleSubmitUI creates a new simple submit UI
func NewSimpleSubmitUI(splog *Splog) *SimpleSubmitUI {
	return &SimpleSubmitUI{splog: splog}
}

func (u *SimpleSubmitUI) ShowStack(lines []string) {
	u.splog.Info("Stack to submit:")
	for _, line := range lines {
		u.splog.Info("%s", line)
	}
	u.splog.Newline()
}

func (u *SimpleSubmitUI) ShowRestackStart() {
	u.splog.Info("Restacking branches before submitting...")
}

func (u *SimpleSubmitUI) ShowRestackComplete() {
	// Nothing needed for simple UI
}

func (u *SimpleSubmitUI) ShowPreparing() {
	// Skip - we'll show progress during actual submission
}

func (u *SimpleSubmitUI) ShowBranchPlan(branchName string, action string, isCurrent bool, skip bool, skipReason string) {
	// Only show if skipping (important info), otherwise we'll show during submission
	if skip {
		displayName := branchName
		if isCurrent {
			displayName = branchName + " (current)"
		}
		u.splog.Info("  ▸ %s %s", ColorDim(displayName), ColorDim("— "+skipReason))
	}
}

func (u *SimpleSubmitUI) ShowNoChanges() {
	u.splog.Info("All PRs up to date.")
}

func (u *SimpleSubmitUI) ShowDryRunComplete() {
	u.splog.Info("Dry run complete.")
}

func (u *SimpleSubmitUI) StartSubmitting(items []SubmitItem) {
	u.items = items
	u.completed = 0
	u.failed = 0
	u.splog.Newline()
	u.splog.Info("Submitting...")
}

func (u *SimpleSubmitUI) UpdateSubmitItem(idx int, status string, url string, err error) {
	if idx >= len(u.items) {
		return
	}

	item := u.items[idx]

	switch status {
	case "submitting":
		action := "Creating"
		if item.Action == "update" {
			action = "Updating"
		}
		u.splog.Info("  ⋯ %s %s...", item.BranchName, action)

	case "done":
		u.completed++
		actionDone := "created"
		if item.Action == "update" {
			actionDone = "updated"
		}
		u.splog.Info("  ✓ %s %s → %s", item.BranchName, actionDone, url)

	case "error":
		u.failed++
		u.splog.Info("  ✗ %s failed: %v", item.BranchName, err)
	}

	u.items[idx].Status = status
	u.items[idx].URL = url
	u.items[idx].Error = err
}

func (u *SimpleSubmitUI) Complete() {
	u.splog.Newline()
	if u.failed > 0 {
		u.splog.Info("Completed: %d, Failed: %d", u.completed, u.failed)
	} else if u.completed > 0 {
		u.splog.Info("✓ All %d PRs submitted successfully", u.completed)
	}
}

// ============================================================================
// TTYSubmitUI - Bubbletea implementation for TTY environments
// ============================================================================

// TTYSubmitUI implements SubmitUI with bubbletea for animated progress
type TTYSubmitUI struct {
	splog         *Splog
	items         []SubmitItem
	program       *tea.Program
	model         *ttySubmitModel
	inSubmitPhase bool
	hasShownStack bool
}

// NewTTYSubmitUI creates a new TTY submit UI
func NewTTYSubmitUI(splog *Splog) *TTYSubmitUI {
	return &TTYSubmitUI{splog: splog}
}

func (u *TTYSubmitUI) ShowStack(lines []string) {
	u.splog.Info("Stack to submit:")
	for _, line := range lines {
		u.splog.Info("%s", line)
	}
	u.splog.Newline()
	u.hasShownStack = true
}

func (u *TTYSubmitUI) ShowRestackStart() {
	u.splog.Info("Restacking branches before submitting...")
}

func (u *TTYSubmitUI) ShowRestackComplete() {
	// Nothing needed
}

func (u *TTYSubmitUI) ShowPreparing() {
	// Skip - we'll show this in the bubbletea UI
}

func (u *TTYSubmitUI) ShowBranchPlan(branchName string, action string, isCurrent bool, skip bool, skipReason string) {
	// Skip - we'll show this in the bubbletea UI during submission
}

func (u *TTYSubmitUI) ShowNoChanges() {
	u.splog.Info("All PRs up to date.")
}

func (u *TTYSubmitUI) ShowDryRunComplete() {
	u.splog.Info("Dry run complete.")
}

func (u *TTYSubmitUI) StartSubmitting(items []SubmitItem) {
	u.items = make([]SubmitItem, len(items))
	copy(u.items, items)
	u.inSubmitPhase = true

	// Add a blank line before starting bubbletea if we showed the stack
	if u.hasShownStack {
		u.splog.Newline()
	}

	u.model = newTTYSubmitModel(u.items)
	u.program = tea.NewProgram(u.model, tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout))

	// Run program in background
	go func() {
		u.program.Run()
	}()
}

func (u *TTYSubmitUI) UpdateSubmitItem(idx int, status string, url string, err error) {
	if !u.inSubmitPhase || u.program == nil {
		return
	}
	u.program.Send(progressUpdateMsg{
		idx:    idx,
		status: status,
		url:    url,
		err:    err,
	})
}

func (u *TTYSubmitUI) Complete() {
	if !u.inSubmitPhase || u.program == nil {
		return
	}
	u.program.Send(progressCompleteMsg{})
	u.program.Wait()
}

// ============================================================================
// Internal bubbletea model for TTY submit progress
// ============================================================================

type ttySubmitModel struct {
	items   []SubmitItem
	spinner spinner.Model
	done    bool
	styles  submitStyles
}

type progressUpdateMsg struct {
	idx    int
	status string
	url    string
	err    error
}

type progressCompleteMsg struct{}

func newTTYSubmitModel(items []SubmitItem) *ttySubmitModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return &ttySubmitModel{
		items:   items,
		spinner: s,
		styles: submitStyles{
			spinnerStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("205")),
			doneStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
			errorStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
			branchStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true),
			urlStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
			dimStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		},
	}
}

func (m *ttySubmitModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m *ttySubmitModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case progressUpdateMsg:
		if msg.idx < len(m.items) {
			m.items[msg.idx].Status = msg.status
			m.items[msg.idx].URL = msg.url
			m.items[msg.idx].Error = msg.err
		}
		return m, m.spinner.Tick

	case progressCompleteMsg:
		m.done = true
		return m, tea.Quit
	}

	return m, nil
}

func (m *ttySubmitModel) View() string {
	var b strings.Builder
	b.WriteString("\n")

	for i, item := range m.items {
		var icon string
		var status string

		switch item.Status {
		case "pending", "":
			icon = m.styles.dimStyle.Render("○")
			// Show the planned action for pending items
			actionPlan := "will create"
			if item.Action == "update" {
				actionPlan = "will update"
			}
			status = m.styles.dimStyle.Render(actionPlan)
		case "submitting":
			icon = m.spinner.View()
			action := "creating"
			if item.Action == "update" {
				action = "updating"
			}
			status = m.styles.spinnerStyle.Render(action + "...")
		case "done":
			icon = m.styles.doneStyle.Render("✓")
			action := "created"
			if item.Action == "update" {
				action = "updated"
			}
			status = m.styles.doneStyle.Render(action)
		case "error":
			icon = m.styles.errorStyle.Render("✗")
			status = m.styles.errorStyle.Render("failed")
		}

		branchName := m.styles.branchStyle.Render(item.BranchName)
		line := fmt.Sprintf("  %s %s %s", icon, branchName, status)

		if item.Status == "done" && item.URL != "" {
			line += " " + m.styles.urlStyle.Render("→ "+item.URL)
		}
		if item.Status == "error" && item.Error != nil {
			line += " " + m.styles.errorStyle.Render(item.Error.Error())
		}

		b.WriteString(line)
		if i < len(m.items)-1 {
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")

	if m.done {
		completed := 0
		failed := 0
		for _, item := range m.items {
			if item.Status == "done" {
				completed++
			} else if item.Status == "error" {
				failed++
			}
		}
		b.WriteString("\n")
		if failed > 0 {
			b.WriteString(m.styles.errorStyle.Render(fmt.Sprintf("Completed: %d, Failed: %d", completed, failed)))
		} else {
			b.WriteString(m.styles.doneStyle.Render(fmt.Sprintf("✓ All %d PRs submitted successfully", completed)))
		}
		b.WriteString("\n")
	}

	return b.String()
}
