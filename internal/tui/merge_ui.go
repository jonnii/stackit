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

// MergeGroup represents a group of steps that should be displayed as a single line
type MergeGroup struct {
	Label       string
	StepIndices []int
}

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
	groups     []MergeGroup
	steps      []MergeStepItem
	currentIdx int
	spinner    spinner.Model
	done       bool
	quitting   bool
	styles     mergeStyles
	updates    <-chan ProgressUpdate
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
func NewMergeTUIModel(groups []MergeGroup, stepDescriptions []string) MergeTUIModel {
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

	// If no groups provided, create one group per step
	if len(groups) == 0 {
		groups = make([]MergeGroup, len(stepDescriptions))
		for i, desc := range stepDescriptions {
			groups[i] = MergeGroup{
				Label:       desc,
				StepIndices: []int{i},
			}
		}
	}

	return MergeTUIModel{
		groups:     groups,
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
	if msg, ok := msg.(tea.KeyMsg); ok {
		if msg.String() == KeyCtrlC || msg.String() == KeyQuit {
			m.quitting = true
			return m, tea.Quit
		}
	}

	switch msg := msg.(type) {
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
				if m.currentIdx == len(m.steps) {
					m.done = true
				}
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

	for i, group := range m.groups {
		var icon string
		var status string
		var groupStatus string
		var activeStep *MergeStepItem
		var failedStep *MergeStepItem

		allDone := true
		allPending := true

		for _, idx := range group.StepIndices {
			step := &m.steps[idx]
			if step.Status == mergeStatusError {
				failedStep = step
				groupStatus = mergeStatusError
				break
			}
			if step.Status != mergeStatusDone {
				allDone = false
			}
			if step.Status != mergeStatusPending {
				allPending = false
			}
			if (step.Status == mergeStatusRunning || step.Status == mergeStatusWaiting) && activeStep == nil {
				activeStep = step
			}
		}

		if groupStatus != mergeStatusError {
			switch {
			case allDone:
				groupStatus = mergeStatusDone
			case allPending:
				groupStatus = mergeStatusPending
			default:
				groupStatus = mergeStatusRunning
			}
		}

		switch groupStatus {
		case mergeStatusPending:
			icon = m.styles.dimStyle.Render("○")
			status = m.styles.dimStyle.Render("pending")
		case mergeStatusRunning:
			icon = m.spinner.View()
			if activeStep != nil {
				if activeStep.Status == mergeStatusWaiting {
					elapsed := activeStep.WaitElapsed.Round(time.Second)
					timeout := activeStep.WaitTimeout.Round(time.Second)
					if timeout == 0 {
						timeout = 10 * time.Minute
					}
					timeStr := fmt.Sprintf("(%v / %v)", elapsed, timeout)
					status = m.styles.waitStyle.Render("waiting for CI...") + " " + m.styles.timeStyle.Render(timeStr)
				} else {
					// Shorten the description if it's redundant with the group label
					desc := activeStep.Description
					if strings.Contains(desc, group.Label) {
						// Extract the action from the description
						// e.g. "Merge PR #101 (branch-a)" -> "merging"
						switch {
						case strings.HasPrefix(desc, "Merge PR"):
							desc = "merging"
						case strings.HasPrefix(desc, "Delete local branch"):
							desc = "deleting local branch"
						case strings.HasPrefix(desc, "Restack"):
							desc = "restacking"
						}
					}
					status = m.styles.spinnerStyle.Render(desc + "...")
				}
			} else {
				status = m.styles.spinnerStyle.Render("running...")
			}
		case mergeStatusDone:
			icon = m.styles.doneStyle.Render("✓")
			status = m.styles.doneStyle.Render("done")
		case mergeStatusError:
			icon = m.styles.errorStyle.Render("✗")
			status = m.styles.errorStyle.Render("failed")
		}

		line := fmt.Sprintf("  %s %d. %s %s", icon, i+1, group.Label, status)

		if groupStatus == mergeStatusError && failedStep != nil && failedStep.Error != nil {
			line += " " + m.styles.errorStyle.Render("→ "+failedStep.Error.Error())
		}

		b.WriteString(line)
		if i < len(m.groups)-1 {
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")

	if m.done {
		completedGroups := 0
		failedGroups := 0
		for _, group := range m.groups {
			groupDone := true
			groupFailed := false
			for _, idx := range group.StepIndices {
				if m.steps[idx].Status == mergeStatusError {
					groupFailed = true
					break
				}
				if m.steps[idx].Status != mergeStatusDone {
					groupDone = false
				}
			}
			if groupFailed {
				failedGroups++
			} else if groupDone {
				completedGroups++
			}
		}
		b.WriteString("\n")
		if failedGroups > 0 {
			b.WriteString(m.styles.errorStyle.Render(fmt.Sprintf("Completed: %d, Failed: %d", completedGroups, failedGroups)))
		} else {
			b.WriteString(m.styles.doneStyle.Render(fmt.Sprintf("✓ All %d steps completed successfully", completedGroups)))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// ProgressUpdate represents an update to merge progress
type ProgressUpdate struct {
	Type        string // "started", "completed", "failed", "waiting"
	StepIndex   int
	Description string
	Error       error
	Elapsed     time.Duration
	Timeout     time.Duration
}

// RunMergeTUI runs the merge TUI with channel-based updates
func RunMergeTUI(groups []MergeGroup, stepDescriptions []string, updates <-chan ProgressUpdate, done chan<- bool) error {
	m := NewMergeTUIModel(groups, stepDescriptions)
	m.updates = updates
	m.doneChan = done

	// Create a program
	program := tea.NewProgram(m, tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout))

	_, err := program.Run()

	// Signal completion after the program has finished and restored the terminal
	if done != nil {
		select {
		case done <- true:
		default:
		}
	}

	return err
}
