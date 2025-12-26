package style

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

// GetLogShortColor returns a styled string with the color from StackitColors
func GetLogShortColor(text string, index int) string {
	if len(StackitColors) == 0 {
		return text
	}

	colorIndex := (index / 2) % len(StackitColors)
	color := StackitColors[colorIndex]

	// Convert RGB to hex color
	hexColor := lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", color[0], color[1], color[2]))

	style := lipgloss.NewStyle().
		Foreground(hexColor)

	return style.Render(text)
}

// FormatShortLine applies color formatting to a short log line
func FormatShortLine(line string, circleIndex, arrowIndex int, isCurrent bool, overallIndent int) string {
	if circleIndex == -1 || arrowIndex == -1 {
		return line
	}

	// Find the arrow character and get its full width in bytes
	arrowRune := '▸'
	arrowWidth := utf8.RuneLen(arrowRune)

	// Split line into parts, skipping the full arrow character
	beforeArrow := line[:arrowIndex]
	afterArrow := line[arrowIndex+arrowWidth:]

	// Color the tree characters before the arrow
	var coloredBefore strings.Builder
	for i, char := range beforeArrow {
		coloredChar := GetLogShortColor(string(char), i)
		// Replace circle if current branch
		if char == '◯' && isCurrent {
			coloredChar = GetLogShortColor("◉", i)
		}
		coloredBefore.WriteString(coloredChar)
	}

	// Color the arrow character
	arrowChar := GetLogShortColor("▸", arrowIndex)

	// Color the branch name and details after the arrow
	coloredAfter := GetLogShortColor(afterArrow, circleIndex)

	// Calculate padding
	padding := overallIndent*2 + 3 - arrowIndex
	if padding > 0 {
		coloredBefore.WriteString(strings.Repeat(" ", padding))
	}

	return coloredBefore.String() + arrowChar + coloredAfter
}

// ColorBranchName colors a branch name based on whether it's current
func ColorBranchName(branchName string, isCurrent bool) string {
	if isCurrent {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("6")).
			Render(branchName + " (current)")
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("12")).
		Render(branchName)
}

// ColorNeedsRestack colors the "needs restack" text
func ColorNeedsRestack(text string) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("3")).
		Render(text)
}

// ColorDim makes text dim/gray
func ColorDim(text string) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render(text)
}

// ColorMagenta colors text magenta
func ColorMagenta(text string) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("5")).
		Render(text)
}

// ColorPRNumber colors a PR number (yellow)
func ColorPRNumber(prNumber int) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("3")).
		Render(fmt.Sprintf("PR #%d", prNumber))
}

// ColorPRState colors PR state text based on state and draft status
func ColorPRState(state string, isDraft bool) string {
	if isDraft {
		return ColorDim("(Draft)")
	}

	switch state {
	case "APPROVED":
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("2")).
			Render("(Approved)")
	case "CHANGES_REQUESTED":
		return ColorMagenta("(Changes Requested)")
	case "REVIEW_REQUIRED":
		return ColorYellow("(Review Required)")
	default:
		// No review decision means review isn't required
		return ""
	}
}

// GetScopeColor returns a deterministic color for a scope string
func GetScopeColor(scope string) (lipgloss.Color, bool) {
	if scope == "" {
		return lipgloss.Color(""), false
	}
	// Simple hash to select from StackitColors
	var hash uint32
	for i := 0; i < len(scope); i++ {
		hash = uint32(scope[i]) + (hash << 6) + (hash << 16) - hash
	}
	colorIndex := int(hash) % len(StackitColors)
	color := StackitColors[colorIndex]
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", color[0], color[1], color[2])), true
}

// ColorScope colors a scope string deterministically
func ColorScope(scope string) string {
	if color, ok := GetScopeColor(scope); ok {
		return lipgloss.NewStyle().Foreground(color).Render("[" + scope + "]")
	}
	return ColorDim("[" + scope + "]")
}
