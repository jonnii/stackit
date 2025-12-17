package output

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SubmitProgressUI defines the interface for submit progress display
type SubmitProgressUI interface {
	// Start initializes the UI with items to submit
	Start(items []SubmitItem)
	// UpdateItem updates the status of a specific item
	UpdateItem(idx int, status string, url string, err error)
	// Complete finalizes the UI and shows summary
	Complete()
}

// NewSubmitProgressUI creates the appropriate progress UI based on TTY availability
func NewSubmitProgressUI(splog *Splog) SubmitProgressUI {
	if IsTTY() {
		return NewTTYSubmitProgress(splog)
	}
	return NewSimpleSubmitProgress(splog)
}

// SimpleSubmitProgress prints progress line by line (non-TTY)
type SimpleSubmitProgress struct {
	splog     *Splog
	items     []SubmitItem
	completed int
	failed    int
}

// NewSimpleSubmitProgress creates a new simple progress UI
func NewSimpleSubmitProgress(splog *Splog) *SimpleSubmitProgress {
	return &SimpleSubmitProgress{splog: splog}
}

func (p *SimpleSubmitProgress) Start(items []SubmitItem) {
	p.items = items
	p.completed = 0
	p.failed = 0
}

func (p *SimpleSubmitProgress) UpdateItem(idx int, status string, url string, err error) {
	if idx >= len(p.items) {
		return
	}

	item := p.items[idx]

	switch status {
	case "submitting":
		action := "Creating"
		if item.Action == "update" {
			action = "Updating"
		}
		p.splog.Info("  ⋯ %s %s...", item.BranchName, action)

	case "done":
		p.completed++
		actionDone := "created"
		if item.Action == "update" {
			actionDone = "updated"
		}
		p.splog.Info("  ✓ %s %s → %s", item.BranchName, actionDone, url)

	case "error":
		p.failed++
		p.splog.Info("  ✗ %s failed: %v", item.BranchName, err)
	}

	p.items[idx].Status = status
	p.items[idx].URL = url
	p.items[idx].Error = err
}

func (p *SimpleSubmitProgress) Complete() {
	p.splog.Newline()
	if p.failed > 0 {
		p.splog.Info("Completed: %d, Failed: %d", p.completed, p.failed)
	} else if p.completed > 0 {
		p.splog.Info("✓ All %d PRs submitted successfully", p.completed)
	}
}

// TTYSubmitProgress uses bubbletea for animated progress (TTY)
type TTYSubmitProgress struct {
	splog   *Splog
	items   []SubmitItem
	program *tea.Program
	model   *ttyProgressModel
}

// NewTTYSubmitProgress creates a new TTY progress UI
func NewTTYSubmitProgress(splog *Splog) *TTYSubmitProgress {
	return &TTYSubmitProgress{splog: splog}
}

func (p *TTYSubmitProgress) Start(items []SubmitItem) {
	p.items = make([]SubmitItem, len(items))
	copy(p.items, items)

	p.model = newTTYProgressModel(p.items)
	p.program = tea.NewProgram(p.model, tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout))

	// Run program in background
	go func() {
		p.program.Run()
	}()
}

func (p *TTYSubmitProgress) UpdateItem(idx int, status string, url string, err error) {
	if p.program == nil {
		return
	}
	p.program.Send(progressUpdateMsg{
		idx:    idx,
		status: status,
		url:    url,
		err:    err,
	})
}

func (p *TTYSubmitProgress) Complete() {
	if p.program == nil {
		return
	}
	p.program.Send(progressCompleteMsg{})
	p.program.Wait()
}

// Internal bubbletea model for TTY progress
type ttyProgressModel struct {
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

func newTTYProgressModel(items []SubmitItem) *ttyProgressModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return &ttyProgressModel{
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

func (m *ttyProgressModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m *ttyProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m *ttyProgressModel) View() string {
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
