package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	keyCtrlC = "ctrl+c"
	keyQuit  = "q"
)

// MergeStepItem represents a step in the merge process
type MergeStepItem struct {
	StepIndex   int
	Description string
	Status      string // "pending", "running", "done", "error", "waiting"
	Error       error
	WaitElapsed time.Duration
	WaitTimeout time.Duration
}

// MergeTUIModel is the bubbletea model for merge progress
type MergeTUIModel struct {
	steps      []MergeStepItem
	currentIdx int
	spinner    spinner.Model
	done       bool
	quitting   bool
	styles     mergeStyles
	updates    <-chan MergeProgressUpdate
	doneChan   chan<- bool
}

type mergeStyles struct {
	spinnerStyle lipgloss.Style
	doneStyle    lipgloss.Style
	errorStyle   lipgloss.Style
	waitStyle    lipgloss.Style
	dimStyle     lipgloss.Style
	timeStyle    lipgloss.Style
}

const (
	mergeStatusPending = "pending"
	mergeStatusRunning = "running"
	mergeStatusWaiting = "waiting"
	mergeStatusDone    = "done"
	mergeStatusError   = "error"
)

// StepUpdateMsg is sent when a step status changes
type StepUpdateMsg struct {
	StepIndex int
	Status    string
	Error     error
}

// StepWaitUpdateMsg is sent to update wait timer
type StepWaitUpdateMsg struct {
	StepIndex int
	Elapsed   time.Duration
	Timeout   time.Duration
}

// NewMergeTUIModel creates a new merge TUI model
func NewMergeTUIModel(stepDescriptions []string) MergeTUIModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	steps := make([]MergeStepItem, len(stepDescriptions))
	for i, desc := range stepDescriptions {
		steps[i] = MergeStepItem{
			StepIndex:   i,
			Description: desc,
			Status:      mergeStatusPending,
		}
	}

	return MergeTUIModel{
		steps:      steps,
		currentIdx: 0,
		spinner:    s,
		styles: mergeStyles{
			spinnerStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("205")),
			doneStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
			errorStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
			waitStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("214")),
			dimStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
			timeStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		},
	}
}

// Init initializes the bubbletea model
func (m MergeTUIModel) Init() tea.Cmd {
	// Start spinner and update ticker
	return tea.Batch(m.spinner.Tick, m.checkForUpdates())
}

// checkForUpdates checks for updates from the channel
func (m MergeTUIModel) checkForUpdates() tea.Cmd {
	if m.updates == nil {
		return nil
	}

	return tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg {
		select {
		case update, ok := <-m.updates:
			if !ok {
				// Channel closed, signal completion
				if m.doneChan != nil {
					select {
					case m.doneChan <- true:
					default:
					}
				}
				return tea.Quit()
			}

			var msg tea.Msg
			switch update.Type {
			case "started":
				msg = StepUpdateMsg{
					StepIndex: update.StepIndex,
					Status:    mergeStatusRunning,
				}
			case "completed":
				msg = StepUpdateMsg{
					StepIndex: update.StepIndex,
					Status:    mergeStatusDone,
				}
			case "failed":
				msg = StepUpdateMsg{
					StepIndex: update.StepIndex,
					Status:    mergeStatusError,
					Error:     update.Error,
				}
			case "waiting":
				msg = StepWaitUpdateMsg{
					StepIndex: update.StepIndex,
					Elapsed:   update.Elapsed,
					Timeout:   update.Timeout,
				}
			}
			return msg
		default:
			// No update available, return nil to continue polling
			return nil
		}
	})
}

// Update handles message updates for the bubbletea model
func (m MergeTUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == keyCtrlC || msg.String() == keyQuit {
			m.quitting = true
			return m, tea.Quit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		// Also check for updates
		return m, tea.Batch(cmd, m.checkForUpdates())

	case StepUpdateMsg:
		if msg.StepIndex < len(m.steps) {
			m.steps[msg.StepIndex].Status = msg.Status
			if msg.Error != nil {
				m.steps[msg.StepIndex].Error = msg.Error
			}
			// If step completed, move to next
			if msg.Status == mergeStatusDone && msg.StepIndex == m.currentIdx {
				m.currentIdx++
			}
			// If step failed, mark as done
			if msg.Status == mergeStatusError {
				m.done = true
			}
		}
		// Continue checking for updates
		return m, m.checkForUpdates()

	case StepWaitUpdateMsg:
		if msg.StepIndex < len(m.steps) {
			m.steps[msg.StepIndex].WaitElapsed = msg.Elapsed
			m.steps[msg.StepIndex].WaitTimeout = msg.Timeout
			m.steps[msg.StepIndex].Status = mergeStatusWaiting
		}
		// Continue checking for updates
		return m, m.checkForUpdates()

	case tea.QuitMsg:
		return m, tea.Quit
	}

	return m, nil
}

// View renders the TUI
func (m MergeTUIModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString("Merge Progress:\n")
	b.WriteString("\n")

	for i, step := range m.steps {
		var icon string
		var status string

		switch step.Status {
		case mergeStatusPending:
			icon = m.styles.dimStyle.Render("○")
			status = m.styles.dimStyle.Render("pending")
		case mergeStatusRunning:
			icon = m.spinner.View()
			status = m.styles.spinnerStyle.Render("running...")
		case mergeStatusWaiting:
			icon = m.spinner.View()
			elapsed := step.WaitElapsed.Round(time.Second)
			timeout := step.WaitTimeout.Round(time.Second)
			if timeout == 0 {
				timeout = 10 * time.Minute // Default display
			}
			timeStr := fmt.Sprintf("(%v / %v)", elapsed, timeout)
			status = m.styles.waitStyle.Render("waiting for CI...") + " " + m.styles.timeStyle.Render(timeStr)
		case mergeStatusDone:
			icon = m.styles.doneStyle.Render("✓")
			status = m.styles.doneStyle.Render("done")
		case mergeStatusError:
			icon = m.styles.errorStyle.Render("✗")
			status = m.styles.errorStyle.Render("failed")
		}

		line := fmt.Sprintf("  %s %d. %s %s", icon, i+1, step.Description, status)

		if step.Status == mergeStatusError && step.Error != nil {
			line += " " + m.styles.errorStyle.Render("→ "+step.Error.Error())
		}

		b.WriteString(line)
		if i < len(m.steps)-1 {
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")

	if m.done {
		completed := 0
		failed := 0
		for _, step := range m.steps {
			if step.Status == mergeStatusDone {
				completed++
			} else if step.Status == mergeStatusError {
				failed++
			}
		}
		b.WriteString("\n")
		if failed > 0 {
			b.WriteString(m.styles.errorStyle.Render(fmt.Sprintf("Completed: %d, Failed: %d", completed, failed)))
		} else {
			b.WriteString(m.styles.doneStyle.Render(fmt.Sprintf("✓ All %d steps completed successfully", completed)))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// MergeProgressUpdate represents an update to merge progress
type MergeProgressUpdate struct {
	Type        string // "started", "completed", "failed", "waiting"
	StepIndex   int
	Description string
	Error       error
	Elapsed     time.Duration
	Timeout     time.Duration
}

// RunMergeTUI runs the merge TUI with channel-based updates
func RunMergeTUI(stepDescriptions []string, updates <-chan MergeProgressUpdate, done chan<- bool) error {
	m := NewMergeTUIModel(stepDescriptions)
	m.updates = updates
	m.doneChan = done

	// Create a program
	program := tea.NewProgram(m, tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout))

	_, err := program.Run()
	return err
}
