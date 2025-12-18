package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
)

// SubmitItem represents a branch being submitted
type SubmitItem struct {
	BranchName string
	Action     string // "create" or "update"
	PRNumber   *int
	Status     string // "pending", "submitting", "done", "error"
	IsSkipped  bool
	SkipReason string
	URL        string
	Error      error
}

// SubmitTUIModel is the bubbletea model for submit progress
type SubmitTUIModel struct {
	items      []SubmitItem
	currentIdx int
	spinner    spinner.Model
	done       bool
	quitting   bool
	submitFunc func(idx int) tea.Cmd
	styles     submitStyles
}

type submitStyles struct {
	spinnerStyle lipgloss.Style
	doneStyle    lipgloss.Style
	errorStyle   lipgloss.Style
	branchStyle  lipgloss.Style
	urlStyle     lipgloss.Style
	dimStyle     lipgloss.Style
}

// SubmitResultMsg is sent when a single submit completes
type SubmitResultMsg struct {
	Idx   int
	URL   string
	Error error
}

// AllDoneMsg signals all submissions are complete
type AllDoneMsg struct{}

// NewSubmitTUIModel creates a new submit TUI model
func NewSubmitTUIModel(items []SubmitItem, submitFunc func(idx int) tea.Cmd) SubmitTUIModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return SubmitTUIModel{
		items:      items,
		currentIdx: 0,
		spinner:    s,
		submitFunc: submitFunc,
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

func (m SubmitTUIModel) Init() tea.Cmd {
	// Start spinner and first submission
	if len(m.items) > 0 {
		m.items[0].Status = "submitting"
		return tea.Batch(m.spinner.Tick, m.submitFunc(0))
	}
	return nil
}

func (m SubmitTUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			m.quitting = true
			return m, tea.Quit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case SubmitResultMsg:
		if msg.Idx < len(m.items) {
			if msg.Error != nil {
				m.items[msg.Idx].Status = "error"
				m.items[msg.Idx].Error = msg.Error
			} else {
				m.items[msg.Idx].Status = "done"
				m.items[msg.Idx].URL = msg.URL
			}
		}

		// Move to next item
		m.currentIdx++
		if m.currentIdx < len(m.items) {
			m.items[m.currentIdx].Status = "submitting"
			return m, tea.Batch(m.spinner.Tick, m.submitFunc(m.currentIdx))
		}

		// All done
		m.done = true
		return m, tea.Quit

	case AllDoneMsg:
		m.done = true
		return m, tea.Quit
	}

	return m, nil
}

func (m SubmitTUIModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n")

	for i, item := range m.items {
		var icon string
		var status string

		switch item.Status {
		case "pending":
			icon = m.styles.dimStyle.Render("○")
			status = m.styles.dimStyle.Render("pending")
		case "submitting":
			icon = m.spinner.View()
			action := "Creating"
			if item.Action == "update" {
				action = "Updating"
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

// IsTTY returns true if we can use a TTY for interactive TUI
func IsTTY() bool {
	// First check if stdin/stdout are terminals
	if !((isatty.IsTerminal(os.Stdin.Fd()) || isatty.IsCygwinTerminal(os.Stdin.Fd())) &&
		(isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd()))) {
		return false
	}
	// Also try to open /dev/tty to verify it's actually available
	f, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return false
	}
	f.Close()
	return true
}

// RunSubmitTUI runs the submit TUI and returns when complete
func RunSubmitTUI(items []SubmitItem, submitFunc func(idx int) tea.Cmd) error {
	m := NewSubmitTUIModel(items, submitFunc)
	// Use WithInput/WithOutput to avoid TTY requirement in non-interactive environments
	p := tea.NewProgram(m, tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout))
	_, err := p.Run()
	return err
}

// RunSubmitTUISimple runs a simple non-interactive version for non-TTY environments
func RunSubmitTUISimple(items []SubmitItem, submitFunc func(idx int) (string, error), splog *Splog) error {
	for i, item := range items {
		action := "Creating"
		if item.Action == "update" {
			action = "Updating"
		}
		splog.Info("  ⋯ %s %s...", item.BranchName, action)

		url, err := submitFunc(i)
		if err != nil {
			splog.Info("  ✗ %s failed: %v", item.BranchName, err)
			return err
		}

		actionDone := "created"
		if item.Action == "update" {
			actionDone = "updated"
		}
		splog.Info("  ✓ %s %s → %s", item.BranchName, actionDone, url)
	}
	return nil
}
