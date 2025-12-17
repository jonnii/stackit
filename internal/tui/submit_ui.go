package tui

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
	// For now, always use simple UI - TTY version can be added later
	return NewSimpleSubmitUI(splog)
}

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
	u.splog.Info("Preparing...")
}

func (u *SimpleSubmitUI) ShowBranchPlan(branchName string, action string, isCurrent bool, skip bool, skipReason string) {
	displayName := branchName
	if isCurrent {
		displayName = branchName + " (current)"
	}

	if skip {
		u.splog.Info("  ▸ %s %s", ColorDim(displayName), ColorDim("— "+skipReason))
	} else {
		u.splog.Info("  ▸ %s → %s", ColorBranchName(branchName, isCurrent), ColorDim(action))
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

