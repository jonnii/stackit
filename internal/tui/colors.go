package tui

import "github.com/charmbracelet/lipgloss"

// STACKIT_COLORS defines the color palette for branch visualization
// Matching the TypeScript version
var STACKIT_COLORS = [][]int{
	{76, 203, 241},  // Light blue
	{77, 202, 125},  // Green
	{110, 173, 38},  // Dark green
	{245, 200, 0},   // Yellow
	{248, 144, 72},  // Orange
	{244, 98, 81},   // Red
	{235, 130, 188}, // Pink
	{159, 131, 228}, // Purple
	{80, 132, 243},  // Blue
}

// ColorRed colors text red
func ColorRed(text string) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("1")).
		Render(text)
}

// ColorYellow colors text yellow
func ColorYellow(text string) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("3")).
		Render(text)
}

// ColorCyan colors text cyan
func ColorCyan(text string) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("6")).
		Render(text)
}
