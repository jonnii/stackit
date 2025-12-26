package submit

import (
	"github.com/charmbracelet/lipgloss"
)

// Item represents a branch being submitted
type Item struct {
	BranchName string
	Action     string // "create" or "update"
	PRNumber   *int
	Status     string // "pending", "submitting", "done", "error"
	IsSkipped  bool
	SkipReason string
	URL        string
	Error      error
}

// Styles defines the visual styling for the submit component
type Styles struct {
	SpinnerStyle lipgloss.Style
	DoneStyle    lipgloss.Style
	ErrorStyle   lipgloss.Style
	BranchStyle  lipgloss.Style
	UrlStyle     lipgloss.Style
	DimStyle     lipgloss.Style
}

// DefaultStyles returns the default styles for the submit component
func DefaultStyles() Styles {
	return Styles{
		SpinnerStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("205")),
		DoneStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
		ErrorStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
		BranchStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true),
		UrlStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		DimStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
	}
}

const (
	StatusSubmitting = "submitting"
	StatusDone       = "done"
	StatusError      = "error"
	StatusPending    = "pending"
)
