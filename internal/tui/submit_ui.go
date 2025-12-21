package tui

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SubmitUI defines the interface for the full submit workflow display
type SubmitUI interface {
	// ShowStack displays the stack to be submitted
	ShowStack(renderer *StackTreeRenderer, rootBranch string)

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
	UpdateSubmitItem(branchName string, status string, url string, err error)

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
	mu        sync.Mutex
}

// NewSimpleSubmitUI creates a new simple submit UI
func NewSimpleSubmitUI(splog *Splog) *SimpleSubmitUI {
	return &SimpleSubmitUI{splog: splog}
}

// ShowStack displays the branch stack being submitted
func (u *SimpleSubmitUI) ShowStack(renderer *StackTreeRenderer, rootBranch string) {
	u.splog.Info("Stack to submit:")
	lines := renderer.RenderStack(rootBranch, TreeRenderOptions{})
	for _, line := range lines {
		u.splog.Info("%s", line)
	}
	u.splog.Newline()
}

// ShowRestackStart indicates the start of the restack process
func (u *SimpleSubmitUI) ShowRestackStart() {
	u.splog.Info("Restacking branches before submitting...")
}

// ShowRestackComplete indicates the completion of the restack process
func (u *SimpleSubmitUI) ShowRestackComplete() {
	// Nothing needed for simple UI
}

// ShowPreparing indicates the preparation phase
func (u *SimpleSubmitUI) ShowPreparing() {
	// Skip - we'll show progress during actual submission
}

// ShowBranchPlan indicates the action planned for a branch
func (u *SimpleSubmitUI) ShowBranchPlan(branchName string, _ string, isCurrent bool, skip bool, skipReason string) {
	// Only show if skipping (important info), otherwise we'll show during submission
	if skip {
		displayName := branchName
		if isCurrent {
			displayName = branchName + " (current)"
		}
		u.splog.Info("  ▸ %s %s", ColorDim(displayName), ColorDim("— "+skipReason))
	}
}

// ShowNoChanges indicates no changes were detected
func (u *SimpleSubmitUI) ShowNoChanges() {
	u.splog.Info("All PRs up to date.")
}

// ShowDryRunComplete indicates completion of a dry run
func (u *SimpleSubmitUI) ShowDryRunComplete() {
	u.splog.Info("Dry run complete.")
}

// StartSubmitting begins the actual submission phase
func (u *SimpleSubmitUI) StartSubmitting(items []SubmitItem) {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.items = items
	u.completed = 0
	u.failed = 0
	u.splog.Newline()
	u.splog.Info("Submitting...")
}

// UpdateSubmitItem updates the status of a specific branch submission
func (u *SimpleSubmitUI) UpdateSubmitItem(branchName string, status string, url string, err error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	var item *SubmitItem
	var itemIdx int
	for i := range u.items {
		if u.items[i].BranchName == branchName {
			item = &u.items[i]
			itemIdx = i
			break
		}
	}

	if item == nil {
		return
	}

	const (
		statusSubmitting = "submitting"
		statusDone       = "done"
		statusError      = "error"
		actionUpdate     = "update"
	)

	switch status {
	case statusSubmitting:
		const (
			labelCreating = "Creating"
			labelUpdating = "Updating"
		)
		action := labelCreating
		if item.Action == actionUpdate {
			action = labelUpdating
		}
		u.splog.Info("  ⋯ %s %s...", item.BranchName, action)

	case statusDone:
		u.completed++
		const (
			actionCreated = "created"
			actionUpdated = "updated"
		)
		actionDone := actionCreated
		if item.Action == actionUpdate {
			actionDone = actionUpdated
		}
		u.splog.Info("  ✓ %s %s → %s", item.BranchName, actionDone, url)

	case statusError:
		u.failed++
		u.splog.Info("  ✗ %s failed: %v", item.BranchName, err)
	}

	u.items[itemIdx].Status = status
	u.items[itemIdx].URL = url
	u.items[itemIdx].Error = err
}

// Complete finalizes the display and shows a summary
func (u *SimpleSubmitUI) Complete() {
	u.mu.Lock()
	defer u.mu.Unlock()

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
	program       *tea.Program
	model         *ttySubmitModel
	inSubmitPhase bool
}

// NewTTYSubmitUI creates a new TTY submit UI
func NewTTYSubmitUI(splog *Splog) *TTYSubmitUI {
	return &TTYSubmitUI{splog: splog}
}

// ShowStack displays the branch stack being submitted
func (u *TTYSubmitUI) ShowStack(renderer *StackTreeRenderer, rootBranch string) {
	u.model = newTTYSubmitModel(nil)
	u.model.renderer = renderer
	u.model.rootBranch = rootBranch

	u.program = tea.NewProgram(u.model, tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout))

	// Run program in background
	go func() {
		if _, err := u.program.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running submit TUI: %v\n", err)
		}
	}()
}

// ShowRestackStart indicates the start of the restack process
func (u *TTYSubmitUI) ShowRestackStart() {
	if u.program != nil {
		u.program.Send(globalMessageMsg("Restacking branches..."))
	}
}

// ShowRestackComplete indicates the completion of the restack process
func (u *TTYSubmitUI) ShowRestackComplete() {
	if u.program != nil {
		u.program.Send(globalMessageMsg(""))
	}
}

// ShowPreparing indicates the preparation phase
func (u *TTYSubmitUI) ShowPreparing() {
	if u.program != nil {
		u.program.Send(globalMessageMsg("Preparing branches..."))
	}
}

// ShowBranchPlan indicates the action planned for a branch
func (u *TTYSubmitUI) ShowBranchPlan(branchName string, action string, isCurrent bool, skip bool, skipReason string) {
	if u.program != nil {
		u.program.Send(planUpdateMsg{
			branchName: branchName,
			action:     action,
			isCurrent:  isCurrent,
			skip:       skip,
			skipReason: skipReason,
		})
	}
}

// ShowNoChanges indicates no changes were detected
func (u *TTYSubmitUI) ShowNoChanges() {
	if u.program != nil {
		u.program.Send(globalMessageMsg("All PRs up to date."))
	}
}

// ShowDryRunComplete indicates completion of a dry run
func (u *TTYSubmitUI) ShowDryRunComplete() {
	if u.program != nil {
		u.program.Send(globalMessageMsg("Dry run complete."))
		u.program.Send(progressCompleteMsg{})
	}
}

// StartSubmitting begins the actual submission phase
func (u *TTYSubmitUI) StartSubmitting(items []SubmitItem) {
	u.inSubmitPhase = true
	if u.program != nil {
		u.program.Send(globalMessageMsg("Submitting..."))
		u.program.Send(startSubmitMsg{items: items})
	}
}

// UpdateSubmitItem updates the status of a specific branch submission
func (u *TTYSubmitUI) UpdateSubmitItem(branchName string, status string, url string, err error) {
	if !u.inSubmitPhase || u.program == nil {
		return
	}
	u.program.Send(progressUpdateMsg{
		branchName: branchName,
		status:     status,
		url:        url,
		err:        err,
	})
}

// Complete finalizes the display and shows a summary
func (u *TTYSubmitUI) Complete() {
	if !u.inSubmitPhase || u.program == nil {
		return
	}
	u.program.Send(globalMessageMsg(""))
	u.program.Send(progressCompleteMsg{})
	u.program.Wait()
}

// ============================================================================
// Internal bubbletea model for TTY submit progress
// ============================================================================

type ttySubmitModel struct {
	items         []SubmitItem
	renderer      *StackTreeRenderer
	rootBranch    string
	spinner       spinner.Model
	done          bool
	styles        submitStyles
	globalMessage string
}

type progressUpdateMsg struct {
	branchName string
	status     string
	url        string
	err        error
}

type startSubmitMsg struct {
	items []SubmitItem
}

type planUpdateMsg struct {
	branchName string
	action     string
	isCurrent  bool
	skip       bool
	skipReason string
}

type globalMessageMsg string

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

	case startSubmitMsg:
		// Update status for items that are in msg.items
		for _, newItem := range msg.items {
			found := false
			for i, item := range m.items {
				if item.BranchName == newItem.BranchName {
					m.items[i].Status = newItem.Status
					m.items[i].Action = newItem.Action
					m.items[i].PRNumber = newItem.PRNumber
					found = true
					break
				}
			}
			if !found {
				m.items = append(m.items, newItem)
			}
		}
		return m, nil

	case planUpdateMsg:
		// Update existing item or add new one
		found := false
		for i, item := range m.items {
			if item.BranchName == msg.branchName {
				m.items[i].Action = msg.action
				m.items[i].IsSkipped = msg.skip
				m.items[i].SkipReason = msg.skipReason
				found = true
				break
			}
		}
		if !found {
			m.items = append(m.items, SubmitItem{
				BranchName: msg.branchName,
				Action:     msg.action,
				IsSkipped:  msg.skip,
				SkipReason: msg.skipReason,
				Status:     "pending",
			})
		}
		return m, nil

	case globalMessageMsg:
		m.globalMessage = string(msg)
		return m, nil

	case progressUpdateMsg:
		for i, item := range m.items {
			if item.BranchName == msg.branchName {
				m.items[i].Status = msg.status
				m.items[i].URL = msg.url
				m.items[i].Error = msg.err
				break
			}
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

	const (
		statusSubmitting = "submitting"
		statusDone       = "done"
		statusError      = "error"
	)

	if m.renderer != nil {
		// Update annotations based on items
		for _, item := range m.items {
			ann := m.renderer.Annotations[item.BranchName]

			// Update PR action if known
			if item.Action != "" {
				ann.PRAction = item.Action
			}

			// Update custom label for status
			if item.IsSkipped {
				ann.CustomLabel = m.styles.dimStyle.Render("(skipped: " + item.SkipReason + ")")
			} else {
				switch item.Status {
				case statusSubmitting:
					ann.CustomLabel = m.styles.spinnerStyle.Render(m.spinner.View() + " submitting...")
				case statusDone:
					ann.CustomLabel = m.styles.doneStyle.Render("✓")
					if item.URL != "" {
						ann.CustomLabel += " " + m.styles.urlStyle.Render("→ "+item.URL)
					}
				case statusError:
					ann.CustomLabel = m.styles.errorStyle.Render("✗")
					if item.Error != nil {
						ann.CustomLabel += " " + m.styles.errorStyle.Render(item.Error.Error())
					}
				}
			}
			m.renderer.SetAnnotation(item.BranchName, ann)
		}

		lines := m.renderer.RenderStack(m.rootBranch, TreeRenderOptions{})
		b.WriteString(strings.Join(lines, "\n"))
	} else {
		// Fallback to list view if no renderer
		for i, item := range m.items {
			var icon string
			var status string

			switch item.Status {
			case "pending", "":
				icon = m.styles.dimStyle.Render("○")
				status = m.styles.dimStyle.Render("will " + item.Action)
			case statusSubmitting:
				icon = m.spinner.View()
				const (
					labelCreating = "Creating"
					labelUpdating = "Updating"
				)
				action := labelCreating
				if item.Action == "update" {
					action = labelUpdating
				}
				status = m.styles.spinnerStyle.Render(action + "ing...")
			case statusDone:
				icon = m.styles.doneStyle.Render("✓")
				status = m.styles.doneStyle.Render(item.Action + "ed")
			case statusError:
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
	}

	if m.globalMessage != "" {
		b.WriteString("\n\n")
		b.WriteString(m.styles.dimStyle.Render(m.globalMessage))
	}

	if m.done {
		completed := 0
		failed := 0
		for _, item := range m.items {
			if item.Status == statusDone {
				completed++
			} else if item.Status == statusError {
				failed++
			}
		}
		b.WriteString("\n\n")
		if failed > 0 {
			b.WriteString(m.styles.errorStyle.Render(fmt.Sprintf("Completed: %d, Failed: %d", completed, failed)))
		} else if completed > 0 {
			b.WriteString(m.styles.doneStyle.Render(fmt.Sprintf("✓ All %d PRs submitted successfully", completed)))
		}
	}

	b.WriteString("\n")
	return b.String()
}
