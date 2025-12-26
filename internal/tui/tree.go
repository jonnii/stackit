package tui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

const (
	// CurrentBranchSymbol is the symbol used for the current branch in tree views
	CurrentBranchSymbol = "◉"
	// BranchSymbol is the symbol used for regular branches in tree views
	BranchSymbol = "◯"
)

// BranchAnnotation holds per-branch display metadata
type BranchAnnotation struct {
	PRNumber     *int
	PRAction     string // "create", "update", "skip", ""
	CheckStatus  string // "PASSING", "FAILING", "PENDING", "NONE", ""
	IsDraft      bool
	NeedsRestack bool
	CustomLabel  string // Additional text to display after branch name
	Scope        string
	CommitCount  int
	LinesAdded   int
	LinesDeleted int
	PRState      string // "OPEN", "MERGED", "CLOSED"
}

// TreeRenderOptions configures rendering behavior
type TreeRenderOptions struct {
	Reverse           bool
	Short             bool
	Steps             *int
	OmitCurrentBranch bool
	NoStyleBranchName bool
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
func (r *StackTreeRenderer) RenderStack(branchName string, opts TreeRenderOptions) []string {
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

		childLines := r.getUpstackInclusiveLines(treeRenderArgs{
			short:             args.short,
			reverse:           args.reverse,
			branchName:        child,
			indentLevel:       childIndent,
			parentScopes:      append(append([]string{}, args.parentScopes...), r.Annotations[args.branchName].Scope),
			steps:             childSteps,
			omitCurrentBranch: args.omitCurrentBranch,
			noStyleBranchName: args.noStyleBranchName,
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
		result = append(result, r.getBranchingLine(numChildren, args.reverse, args.indentLevel, args.parentScopes))
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

func (r *StackTreeRenderer) getBranchingLine(numChildren int, reverse bool, indentLevel int, parentScopes []string) string {
	if numChildren < 2 {
		return ""
	}

	var prefixBuilder strings.Builder
	for i := 0; i < indentLevel; i++ {
		scope := ""
		if i < len(parentScopes) {
			scope = parentScopes[i]
		}
		prefixBuilder.WriteString(lipgloss.NewStyle().Foreground(GetScopeColor(scope)).Render("│") + "  ")
	}
	prefix := prefixBuilder.String()

	var middle, last string
	// Use current branch scope for branching characters
	scope := ""
	if indentLevel < len(parentScopes) {
		scope = parentScopes[indentLevel]
	}
	style := lipgloss.NewStyle().Foreground(GetScopeColor(scope))

	if reverse {
		middle = "──┬"
		last = "──┐"
	} else {
		middle = "──┴"
		last = "──┘"
	}

	line := prefix + style.Render("├")
	if numChildren > 2 {
		line += style.Render(strings.Repeat(middle, numChildren-2))
	}
	line += style.Render(last)

	return line
}

func (r *StackTreeRenderer) getInfoLines(args treeRenderArgs) []string {
	isCurrent := args.branchName == r.currentBranch
	annotation := r.Annotations[args.branchName]
	isTrunk := r.isTrunk(args.branchName)
	isMerged := annotation.PRState == "MERGED"
	isClosed := annotation.PRState == "CLOSED"
	isDim := isMerged || isClosed

	// Get branch info with colors
	branchName := args.branchName
	coloredBranchName := ColorBranchName(branchName, isCurrent)

	// Add annotation
	coloredBranchName += r.formatAnnotationColored(annotation)

	// Add compact stats
	coloredBranchName += " " + r.formatCompactStats(annotation, isTrunk)

	// Add restack indicator if needed
	if !r.isBranchFixed(branchName) {
		coloredBranchName += " " + ColorNeedsRestack("(needs restack)")
	}

	if isDim {
		coloredBranchName = ColorDim(coloredBranchName)
	}

	var result []string
	var prefixBuilder strings.Builder
	for i := 0; i < args.indentLevel; i++ {
		scope := ""
		if i < len(args.parentScopes) {
			scope = args.parentScopes[i]
		}
		prefixBuilder.WriteString(lipgloss.NewStyle().Foreground(GetScopeColor(scope)).Render("│") + "  ")
	}
	prefix := prefixBuilder.String()

	var symbol string
	if isCurrent {
		symbol = CurrentBranchSymbol
	} else {
		symbol = BranchSymbol
	}

	// Color the symbol and current branch line based on its own scope
	scope := annotation.Scope
	style := lipgloss.NewStyle().Foreground(GetScopeColor(scope))
	if isDim {
		style = style.Foreground(lipgloss.Color("8"))
	}

	result = append(result, prefix+style.Render(symbol)+" "+coloredBranchName)

	// Add trailing line
	result = append(result, prefix+style.Render("│"))

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

func (r *StackTreeRenderer) formatCompactStats(annotation BranchAnnotation, isTrunk bool) string {
	var parts []string

	if annotation.PRNumber != nil {
		parts = append(parts, fmt.Sprintf("#%d", *annotation.PRNumber))
	}

	if !isTrunk {
		parts = append(parts, fmt.Sprintf("%d commits", annotation.CommitCount))
		if annotation.LinesAdded > 0 || annotation.LinesDeleted > 0 {
			parts = append(parts, fmt.Sprintf("+%d/-%d", annotation.LinesAdded, annotation.LinesDeleted))
		}
	}

	if len(parts) == 0 {
		return ""
	}

	return ColorDim("[" + strings.Join(parts, " • ") + "]")
}

func (r *StackTreeRenderer) formatAnnotationColored(annotation BranchAnnotation) string {
	var parts []string

	if annotation.Scope != "" {
		parts = append(parts, ColorScope(annotation.Scope))
	}

	if annotation.PRAction != "" {
		parts = append(parts, ColorDim("→ "+annotation.PRAction))
	}

	if annotation.CheckStatus != "" && annotation.CheckStatus != "NONE" {
		icon := r.checksIcon(annotation.CheckStatus)
		switch annotation.CheckStatus {
		case "PASSING":
			parts = append(parts, ColorCyan(icon))
		case "FAILING":
			parts = append(parts, ColorRed(icon))
		case "PENDING":
			parts = append(parts, ColorYellow(icon))
		default:
			parts = append(parts, icon)
		}
	}

	if annotation.IsDraft {
		parts = append(parts, ColorDim("(Draft)"))
	}

	if annotation.PRState == "MERGED" {
		parts = append(parts, ColorDim("(Merged)"))
	} else if annotation.PRState == "CLOSED" {
		parts = append(parts, ColorDim("(Closed)"))
	}

	if annotation.CustomLabel != "" {
		parts = append(parts, ColorDim(annotation.CustomLabel))
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
	return "#" + strings.TrimPrefix(ColorPRNumber(prNumber), "PR ")
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

			formatted := FormatShortLine(line, circleIndex, arrowIndex, isCurrent, overallIndent)
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

		line += ColorBranchName(branchName, isCurrent)
		line += r.formatAnnotationColored(annotation)

		result = append(result, line)
	}

	return result
}
