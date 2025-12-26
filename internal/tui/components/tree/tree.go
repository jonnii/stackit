// Package tree provides a renderer for branch tree visualizations.
package tree

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"

	"stackit.dev/stackit/internal/tui/style"
)

const (
	// CurrentBranchSymbol is the symbol used for the current branch in tree views
	CurrentBranchSymbol = "◉"
	// BranchSymbol is the symbol used for regular branches in tree views
	BranchSymbol = "◯"

	// PRStateMerged indicates the PR has been merged
	PRStateMerged = "MERGED"
	// PRStateClosed indicates the PR has been closed
	PRStateClosed = "CLOSED"
)

// BranchAnnotation holds per-branch display metadata
type BranchAnnotation struct {
	PRNumber      *int
	PRAction      string // "create", "update", "skip", ""
	CheckStatus   string // "PASSING", "FAILING", "PENDING", "NONE", ""
	IsDraft       bool
	NeedsRestack  bool
	CustomLabel   string // Additional text to display after branch name
	Scope         string
	ExplicitScope string

	CommitCount  int
	LinesAdded   int
	LinesDeleted int
	PRState      string // "OPEN", "MERGED", "CLOSED"
}

// RenderOptions configures rendering behavior
type RenderOptions struct {
	Reverse           bool
	Short             bool
	Steps             *int
	OmitCurrentBranch bool
	NoStyleBranchName bool
	HideStats         bool
}

// StackTreeRenderer renders branch trees with annotations
type StackTreeRenderer struct {
	currentBranch string
	trunk         string
	getChildren   func(branchName string) []string
	getParent     func(branchName string) string
	isTrunk       func(branchName string) bool
	isBranchFixed func(branchName string) bool
	Annotations   map[string]BranchAnnotation
}

// NewStackTreeRenderer creates a new tree renderer
func NewStackTreeRenderer(
	currentBranch string,
	trunk string,
	getChildren func(branchName string) []string,
	getParent func(branchName string) string,
	isTrunk func(branchName string) bool,
	isBranchFixed func(branchName string) bool,
) *StackTreeRenderer {
	return &StackTreeRenderer{
		currentBranch: currentBranch,
		trunk:         trunk,
		getChildren:   getChildren,
		getParent:     getParent,
		isTrunk:       isTrunk,
		isBranchFixed: isBranchFixed,
		Annotations:   make(map[string]BranchAnnotation),
	}
}

// SetAnnotation sets the annotation for a branch
func (r *StackTreeRenderer) SetAnnotation(branchName string, annotation BranchAnnotation) {
	r.Annotations[branchName] = annotation
}

// SetAnnotations sets annotations for multiple branches
func (r *StackTreeRenderer) SetAnnotations(annotations map[string]BranchAnnotation) {
	r.Annotations = annotations
}

// RenderStack renders the full stack tree starting from a branch
func (r *StackTreeRenderer) RenderStack(branchName string, opts RenderOptions) []string {
	overallIndent := 0
	args := treeRenderArgs{
		short:             opts.Short,
		reverse:           opts.Reverse,
		branchName:        branchName,
		indentLevel:       0,
		parentScopes:      []string{},
		steps:             opts.Steps,
		omitCurrentBranch: opts.OmitCurrentBranch,
		noStyleBranchName: opts.NoStyleBranchName,
		hideStats:         opts.HideStats,
		overallIndent:     &overallIndent,
	}

	outputDeep := [][]string{
		r.getUpstackExclusiveLines(args),
		r.getBranchLines(args),
		r.getDownstackExclusiveLines(args),
	}

	// Reverse if needed
	if opts.Reverse {
		for i, j := 0, len(outputDeep)-1; i < j; i, j = i+1, j-1 {
			outputDeep[i], outputDeep[j] = outputDeep[j], outputDeep[i]
		}
	}

	// Flatten
	var result []string
	for _, section := range outputDeep {
		result = append(result, section...)
	}

	// Apply short formatting if needed
	if opts.Short {
		return r.formatShortLines(result, args)
	}

	return result
}

type treeRenderArgs struct {
	short             bool
	reverse           bool
	branchName        string
	indentLevel       int
	parentScopes      []string
	steps             *int
	omitCurrentBranch bool
	noStyleBranchName bool
	hideStats         bool
	skipBranchingLine bool
	overallIndent     *int
}

func (r *StackTreeRenderer) getUpstackExclusiveLines(args treeRenderArgs) []string {
	if args.steps != nil && *args.steps == 0 {
		return []string{}
	}

	children := r.getChildren(args.branchName)

	// Filter out current branch if needed
	filteredChildren := []string{}
	for _, child := range children {
		if !args.omitCurrentBranch || child != r.currentBranch {
			filteredChildren = append(filteredChildren, child)
		}
	}

	numChildren := len(filteredChildren)
	var result []string

	for i, child := range filteredChildren {
		childSteps := args.steps
		if childSteps != nil {
			nextStep := *childSteps - 1
			childSteps = &nextStep
		}

		var childIndent int
		if args.reverse {
			childIndent = args.indentLevel + (numChildren - i - 1)
		} else {
			childIndent = args.indentLevel + i
		}

		childParentScopes := append([]string{}, args.parentScopes...)
		parentScope := r.Annotations[args.branchName].Scope

		// Fill parentScopes up to childIndent with parentScope
		// This ensures vertical lines for siblings use the parent's scope color
		for len(childParentScopes) < childIndent {
			childParentScopes = append(childParentScopes, parentScope)
		}

		childLines := r.getUpstackInclusiveLines(treeRenderArgs{
			short:             args.short,
			reverse:           args.reverse,
			branchName:        child,
			indentLevel:       childIndent,
			parentScopes:      childParentScopes,
			steps:             childSteps,
			omitCurrentBranch: args.omitCurrentBranch,
			noStyleBranchName: args.noStyleBranchName,
			hideStats:         args.hideStats,
			overallIndent:     args.overallIndent,
		})

		result = append(result, childLines...)
	}

	return result
}

func (r *StackTreeRenderer) getUpstackInclusiveLines(args treeRenderArgs) []string {
	outputDeep := [][]string{
		r.getUpstackExclusiveLines(args),
		r.getBranchLines(args),
	}

	if args.reverse {
		for i, j := 0, len(outputDeep)-1; i < j; i, j = i+1, j-1 {
			outputDeep[i], outputDeep[j] = outputDeep[j], outputDeep[i]
		}
	}

	var result []string
	for _, section := range outputDeep {
		result = append(result, section...)
	}

	return result
}

func (r *StackTreeRenderer) getDownstackExclusiveLines(args treeRenderArgs) []string {
	if r.isTrunk(args.branchName) {
		return []string{}
	}

	// Build stack from current to trunk
	var fullStack []string
	current := args.branchName
	for {
		parent := r.getParent(current)
		if parent == "" || r.isTrunk(parent) {
			break
		}
		fullStack = append([]string{parent}, fullStack...)
		current = parent
	}

	// Prepend trunk
	fullStack = append([]string{r.trunk}, fullStack...)

	// Apply steps limit
	if args.steps != nil && *args.steps > 0 {
		start := len(fullStack) - *args.steps
		if start < 0 {
			start = 0
		}
		fullStack = fullStack[start:]
	}

	var result []string
	for _, branchName := range fullStack {
		branchLines := r.getBranchLines(treeRenderArgs{
			short:             args.short,
			reverse:           args.reverse,
			branchName:        branchName,
			indentLevel:       args.indentLevel,
			parentScopes:      args.parentScopes,
			skipBranchingLine: true,
			overallIndent:     args.overallIndent,
		})
		result = append(result, branchLines...)
	}

	// Reverse if needed (opposite of normal because we got list from trunk upward)
	if !args.reverse {
		for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
			result[i], result[j] = result[j], result[i]
		}
	}

	return result
}

func (r *StackTreeRenderer) getBranchLines(args treeRenderArgs) []string {
	children := r.getChildren(args.branchName)
	numChildren := len(children)

	if args.overallIndent != nil {
		if args.indentLevel > *args.overallIndent {
			*args.overallIndent = args.indentLevel
		}
	}

	// Short format
	if args.short {
		line := strings.Repeat("│ ", args.indentLevel)

		// Add branching characters
		if !args.skipBranchingLine && numChildren > 1 {
			if args.reverse {
				line += strings.Repeat("─┬", numChildren-2) + "─┐"
			} else {
				line += strings.Repeat("─┴", numChildren-2) + "─┘"
			}
		} else if !args.skipBranchingLine && numChildren == 1 {
			if args.reverse {
				line += "─┐"
			} else {
				line += "─┘"
			}
		}

		// Add circle and branch name
		isCurrent := args.branchName == r.currentBranch
		if isCurrent && !args.noStyleBranchName {
			line += CurrentBranchSymbol
		} else {
			line += BranchSymbol
		}
		line += "▸" + args.branchName

		// Add annotation
		annotation := r.Annotations[args.branchName]
		line += r.formatAnnotation(annotation, args.noStyleBranchName)

		// Add restack indicator
		if !args.noStyleBranchName && !r.isBranchFixed(args.branchName) {
			line += " (needs restack)"
		}

		return []string{line}
	}

	// Full format
	var result []string

	// Branching line
	if !args.skipBranchingLine && numChildren >= 2 {
		result = append(result, r.getBranchingLine(numChildren, args.reverse, args.indentLevel, args.parentScopes, args.branchName))
	}

	// Branch info lines
	infoLines := r.getInfoLines(args)
	result = append(result, infoLines...)

	if args.reverse {
		for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
			result[i], result[j] = result[j], result[i]
		}
	}

	return result
}

func (r *StackTreeRenderer) getBranchingLine(numChildren int, reverse bool, indentLevel int, parentScopes []string, branchName string) string {
	if numChildren < 2 {
		return ""
	}

	var prefixBuilder strings.Builder
	for i := 0; i < indentLevel; i++ {
		scope := ""
		if i < len(parentScopes) {
			scope = parentScopes[i]
		}
		char := "│"
		if color, ok := style.GetScopeColor(scope); ok {
			char = lipgloss.NewStyle().Foreground(color).Render(char)
		}
		prefixBuilder.WriteString(char + "  ")
	}
	prefix := prefixBuilder.String()

	var middle, last string
	// The branching characters connect the current branch to its children.
	// They should use the current branch's scope color.
	annotation := r.Annotations[branchName]
	scope := annotation.Scope
	isMerged := annotation.PRState == PRStateMerged
	isClosed := annotation.PRState == PRStateClosed
	isDim := isMerged || isClosed

	styleObj := lipgloss.NewStyle()
	if color, ok := style.GetScopeColor(scope); ok {
		styleObj = styleObj.Foreground(color)
	}

	if isDim {
		styleObj = styleObj.Foreground(lipgloss.Color("8"))
	}

	if reverse {
		middle = "──┬"
		last = "──┐"
	} else {
		middle = "──┴"
		last = "──┘"
	}

	branchingChars := "├"
	if numChildren > 2 {
		branchingChars += strings.Repeat(middle, numChildren-2)
	}
	branchingChars += last

	line := prefix + styleObj.Render(branchingChars)

	return line
}

func (r *StackTreeRenderer) getInfoLines(args treeRenderArgs) []string {
	isCurrent := args.branchName == r.currentBranch
	annotation := r.Annotations[args.branchName]
	isTrunk := r.isTrunk(args.branchName)
	isMerged := annotation.PRState == PRStateMerged
	isClosed := annotation.PRState == PRStateClosed
	isDim := isMerged || isClosed

	// Get branch info with colors
	branchName := args.branchName
	coloredBranchName := style.ColorBranchName(branchName, isCurrent)

	// Add annotation
	coloredBranchName += r.FormatAnnotationColored(annotation)

	// Add compact stats
	coloredBranchName += " " + r.formatCompactStats(annotation, isTrunk, args.hideStats)

	// Add restack indicator if needed
	if !r.isBranchFixed(branchName) {
		coloredBranchName += " " + style.ColorNeedsRestack("(needs restack)")
	}

	if isDim {
		coloredBranchName = style.ColorDim(coloredBranchName)
	}

	var result []string
	var prefixBuilder strings.Builder
	for i := 0; i < args.indentLevel; i++ {
		scope := ""
		if i < len(args.parentScopes) {
			scope = args.parentScopes[i]
		}
		char := "│"
		if color, ok := style.GetScopeColor(scope); ok {
			char = lipgloss.NewStyle().Foreground(color).Render(char)
		}
		prefixBuilder.WriteString(char + "  ")
	}
	prefix := prefixBuilder.String()

	var symbol string
	if isCurrent {
		symbol = CurrentBranchSymbol
	} else {
		symbol = BranchSymbol
	}

	// Style for the branch symbol and name
	styleObj := lipgloss.NewStyle()
	if color, ok := style.GetScopeColor(annotation.Scope); ok {
		styleObj = styleObj.Foreground(color)
	}

	// Style for the vertical line below the symbol (connecting to parent)
	// It should use the parent's scope color, not the branch's own scope.
	parentScope := ""
	if parent := r.getParent(args.branchName); parent != "" {
		parentScope = r.Annotations[parent].Scope
	}
	parentStyle := lipgloss.NewStyle()
	if color, ok := style.GetScopeColor(parentScope); ok {
		parentStyle = parentStyle.Foreground(color)
	}

	if isDim {
		styleObj = styleObj.Foreground(lipgloss.Color("8"))
		parentStyle = parentStyle.Foreground(lipgloss.Color("8"))
	}

	result = append(result, prefix+styleObj.Render(symbol)+" "+coloredBranchName)

	// Add trailing line
	result = append(result, prefix+parentStyle.Render("│"))

	return result
}

func (r *StackTreeRenderer) formatAnnotation(annotation BranchAnnotation, _ bool) string {
	var parts []string

	if annotation.PRNumber != nil {
		parts = append(parts, formatPRNumberPlain(*annotation.PRNumber))
	}

	if annotation.Scope != "" {
		parts = append(parts, "["+annotation.Scope+"]")
	}

	if annotation.PRAction != "" {
		parts = append(parts, annotation.PRAction)
	}

	if annotation.CheckStatus != "" && annotation.CheckStatus != "NONE" {
		icon := r.checksIcon(annotation.CheckStatus)
		parts = append(parts, icon)
	}

	if annotation.IsDraft {
		parts = append(parts, "(Draft)")
	}

	if annotation.CustomLabel != "" {
		parts = append(parts, annotation.CustomLabel)
	}

	if len(parts) == 0 {
		return ""
	}

	return " " + strings.Join(parts, " ")
}

func (r *StackTreeRenderer) formatCompactStats(annotation BranchAnnotation, isTrunk bool, hideStats bool) string {
	var parts []string

	if annotation.PRNumber != nil {
		parts = append(parts, fmt.Sprintf("#%d", *annotation.PRNumber))
	}

	if !isTrunk && !hideStats {
		parts = append(parts, fmt.Sprintf("%d commits", annotation.CommitCount))
		if annotation.LinesAdded > 0 || annotation.LinesDeleted > 0 {
			parts = append(parts, fmt.Sprintf("+%d/-%d", annotation.LinesAdded, annotation.LinesDeleted))
		}
	}

	if len(parts) == 0 {
		return ""
	}

	return style.ColorDim("[" + strings.Join(parts, " • ") + "]")
}

// FormatAnnotationColored returns a colored string representation of a branch annotation
func (r *StackTreeRenderer) FormatAnnotationColored(annotation BranchAnnotation) string {
	var parts []string

	if annotation.ExplicitScope != "" {
		parts = append(parts, style.ColorScope(annotation.ExplicitScope))
	}

	if annotation.PRAction != "" {
		parts = append(parts, style.ColorDim("→ "+annotation.PRAction))
	}

	if annotation.CheckStatus != "" && annotation.CheckStatus != "NONE" {
		icon := r.checksIcon(annotation.CheckStatus)
		switch annotation.CheckStatus {
		case "PASSING":
			parts = append(parts, style.ColorCyan(icon))
		case "FAILING":
			parts = append(parts, style.ColorRed(icon))
		case "PENDING":
			parts = append(parts, style.ColorYellow(icon))
		default:
			parts = append(parts, icon)
		}
	}

	if annotation.IsDraft {
		parts = append(parts, style.ColorDim("(Draft)"))
	}

	if annotation.PRState == PRStateMerged {
		parts = append(parts, style.ColorDim("(Merged)"))
	} else if annotation.PRState == PRStateClosed {
		parts = append(parts, style.ColorDim("(Closed)"))
	}

	if annotation.CustomLabel != "" {
		parts = append(parts, style.ColorDim(annotation.CustomLabel))
	}

	if len(parts) == 0 {
		return ""
	}

	return " " + strings.Join(parts, " ")
}

func (r *StackTreeRenderer) checksIcon(status string) string {
	switch status {
	case "PASSING":
		return "✓"
	case "FAILING":
		return "✗"
	case "PENDING":
		return "⏳"
	default:
		return ""
	}
}

func formatPRNumberPlain(prNumber int) string {
	return "#" + strings.TrimPrefix(style.ColorPRNumber(prNumber), "PR ")
}

func (r *StackTreeRenderer) formatShortLines(lines []string, args treeRenderArgs) []string {
	var result []string

	for _, line := range lines {
		circleIndex := strings.Index(line, BranchSymbol)
		arrowIndex := strings.Index(line, "▸")

		if circleIndex == -1 {
			circleIndex = strings.Index(line, CurrentBranchSymbol)
		}

		if circleIndex != -1 && arrowIndex != -1 {
			// Extract branch name to check if it's current
			// arrowIndex is a byte index, need to skip full UTF-8 character
			arrowRune := '▸'
			arrowWidth := utf8.RuneLen(arrowRune)
			branchNameAndDetails := line[arrowIndex+arrowWidth:]
			branchName := strings.Fields(branchNameAndDetails)[0]
			isCurrent := !args.noStyleBranchName && r.currentBranch != "" && branchName == r.currentBranch

			overallIndent := 0
			if args.overallIndent != nil {
				overallIndent = *args.overallIndent
			}

			formatted := style.FormatShortLine(line, circleIndex, arrowIndex, isCurrent, overallIndent)
			result = append(result, formatted)
		} else {
			result = append(result, line)
		}
	}

	return result
}

// RenderBranchList renders a simple list of branches with annotations (no tree structure)
func (r *StackTreeRenderer) RenderBranchList(branches []string) []string {
	result := make([]string, 0, len(branches))

	for _, branchName := range branches {
		isCurrent := branchName == r.currentBranch
		annotation := r.Annotations[branchName]

		line := "  "
		if isCurrent {
			line += CurrentBranchSymbol + " "
		} else {
			line += BranchSymbol + " "
		}

		line += style.ColorBranchName(branchName, isCurrent)
		line += r.FormatAnnotationColored(annotation)

		result = append(result, line)
	}

	return result
}
