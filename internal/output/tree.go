package output

import (
	"strings"
)

// BranchAnnotation holds per-branch display metadata
type BranchAnnotation struct {
	PRNumber     *int
	PRAction     string // "create", "update", "skip", ""
	CheckStatus  string // "PASSING", "FAILING", "PENDING", "NONE", ""
	IsDraft      bool
	NeedsRestack bool
	CustomLabel  string // Additional text to display after branch name
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
	annotations   map[string]BranchAnnotation
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
		annotations:   make(map[string]BranchAnnotation),
	}
}

// SetAnnotation sets the annotation for a branch
func (r *StackTreeRenderer) SetAnnotation(branchName string, annotation BranchAnnotation) {
	r.annotations[branchName] = annotation
}

// SetAnnotations sets annotations for multiple branches
func (r *StackTreeRenderer) SetAnnotations(annotations map[string]BranchAnnotation) {
	r.annotations = annotations
}

// RenderStack renders the full stack tree starting from a branch
func (r *StackTreeRenderer) RenderStack(branchName string, opts TreeRenderOptions) []string {
	overallIndent := 0
	args := treeRenderArgs{
		short:             opts.Short,
		reverse:           opts.Reverse,
		branchName:        branchName,
		indentLevel:       0,
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

		childIndent := args.indentLevel
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
			line += "◉"
		} else {
			line += "◯"
		}
		line += "▸" + args.branchName

		// Add annotation
		annotation := r.annotations[args.branchName]
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
		result = append(result, r.getBranchingLine(numChildren, args.reverse, args.indentLevel))
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

func (r *StackTreeRenderer) getBranchingLine(numChildren int, reverse bool, indentLevel int) string {
	if numChildren < 2 {
		return ""
	}

	prefix := strings.Repeat("│  ", indentLevel)

	var middle, last string
	if reverse {
		middle = "──┬"
		last = "──┐"
	} else {
		middle = "──┴"
		last = "──┘"
	}

	line := prefix + "├"
	if numChildren > 2 {
		line += strings.Repeat(middle, numChildren-2)
	}
	line += last

	return line
}

func (r *StackTreeRenderer) getInfoLines(args treeRenderArgs) []string {
	isCurrent := args.branchName == r.currentBranch

	// Get branch info with colors
	branchName := args.branchName
	coloredBranchName := ColorBranchName(branchName, isCurrent)

	// Add annotation
	annotation := r.annotations[branchName]
	coloredBranchName += r.formatAnnotationColored(annotation)

	// Add restack indicator if needed
	if !r.isBranchFixed(branchName) {
		coloredBranchName += " " + ColorNeedsRestack("(needs restack)")
	}

	var result []string
	prefix := strings.Repeat("│  ", args.indentLevel)

	var symbol string
	if isCurrent {
		symbol = "◉"
	} else {
		symbol = "◯"
	}

	result = append(result, prefix+symbol+" "+coloredBranchName)

	// Add trailing line
	result = append(result, prefix+"│")

	return result
}

func (r *StackTreeRenderer) formatAnnotation(annotation BranchAnnotation, noStyle bool) string {
	var parts []string

	if annotation.PRNumber != nil {
		parts = append(parts, formatPRNumberPlain(*annotation.PRNumber))
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

func (r *StackTreeRenderer) formatAnnotationColored(annotation BranchAnnotation) string {
	var parts []string

	if annotation.PRNumber != nil {
		parts = append(parts, ColorPRNumber(*annotation.PRNumber))
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
		circleIndex := strings.Index(line, "◯")
		arrowIndex := strings.Index(line, "▸")

		if circleIndex == -1 {
			circleIndex = strings.Index(line, "◉")
		}

		if circleIndex != -1 && arrowIndex != -1 {
			// Extract branch name to check if it's current
			branchNameAndDetails := line[arrowIndex+1:]
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
	var result []string

	for _, branchName := range branches {
		isCurrent := branchName == r.currentBranch
		annotation := r.annotations[branchName]

		line := "  "
		if isCurrent {
			line += "◉ "
		} else {
			line += "◯ "
		}

		line += ColorBranchName(branchName, isCurrent)
		line += r.formatAnnotationColored(annotation)

		result = append(result, line)
	}

	return result
}
