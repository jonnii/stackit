package output

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// GetLogShortColor returns a styled string with the color from STACKIT_COLORS
func GetLogShortColor(text string, index int) string {
	if len(STACKIT_COLORS) == 0 {
		return text
	}

	colorIndex := (index / 2) % len(STACKIT_COLORS)
	color := STACKIT_COLORS[colorIndex]

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

	// Split line into parts
	beforeArrow := line[:arrowIndex]
	afterArrow := line[arrowIndex+1:]

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

	// Color the branch name and details after the arrow
	coloredAfter := GetLogShortColor(afterArrow, circleIndex)

	// Calculate padding
	padding := overallIndent*2 + 3 - arrowIndex
	if padding > 0 {
		coloredBefore.WriteString(strings.Repeat(" ", padding))
	}

	return coloredBefore.String() + coloredAfter
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

