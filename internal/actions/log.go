package actions

import (
	"fmt"
	"strings"

	"stackit.dev/stackit/internal/context"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/output"
)

// LogOptions specifies options for the log command
type LogOptions struct {
	Style        string // "SHORT" or "FULL"
	Reverse      bool
	Steps        *int
	BranchName   string
	ShowUntracked bool
}

// LogAction displays the branch tree
func LogAction(opts LogOptions, ctx *context.Context) error {
	// Populate remote SHAs if needed
	if err := ctx.Engine.PopulateRemoteShas(); err != nil {
		return fmt.Errorf("failed to populate remote SHAs: %w", err)
	}

	// Get stack lines
	stackLines := getStackLines(printStackArgs{
		short:         opts.Style == "SHORT",
		reverse:       opts.Reverse,
		branchName:    opts.BranchName,
		indentLevel:   0,
		steps:         opts.Steps,
		omitCurrentBranch: false,
		noStyleBranchName: false,
	}, ctx)

	// Add untracked branches if requested
	if opts.ShowUntracked {
		untracked := getUntrackedBranchNames(ctx)
		if len(untracked) > 0 {
			stackLines = append(stackLines, "")
			stackLines = append(stackLines, "Untracked branches:")
			stackLines = append(stackLines, untracked...)
		}
	}

	// Output the result
	ctx.Splog.Page(strings.Join(stackLines, "\n"))
	ctx.Splog.Newline()

	return nil
}

type printStackArgs struct {
	short            bool
	reverse          bool
	branchName       string
	indentLevel      int
	steps            *int
	omitCurrentBranch bool
	noStyleBranchName bool
	skipBranchingLine bool
	overallIndent    *int
}

func getStackLines(args printStackArgs, ctx *context.Context) []string {
	overallIndent := 0
	if args.overallIndent == nil {
		args.overallIndent = &overallIndent
	}

	outputDeep := [][]string{
		getUpstackExclusiveLines(printStackArgs{
			short:         args.short,
			reverse:       args.reverse,
			branchName:    args.branchName,
			indentLevel:   args.indentLevel,
			steps:         args.steps,
			omitCurrentBranch: args.omitCurrentBranch,
			noStyleBranchName: args.noStyleBranchName,
			overallIndent: args.overallIndent,
		}, ctx),
		getBranchLines(args, ctx),
		getDownstackExclusiveLines(args, ctx),
	}

	// Reverse if needed
	if args.reverse {
		// Reverse the order of the three sections
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
	if args.short {
		return formatShortLines(result, args, ctx)
	}

	return result
}

func getUpstackExclusiveLines(args printStackArgs, ctx *context.Context) []string {
	if args.steps != nil && *args.steps == 0 {
		return []string{}
	}

	children := ctx.Engine.GetChildren(args.branchName)
	
	// Filter out current branch if needed
	filteredChildren := []string{}
	for _, child := range children {
		if !args.omitCurrentBranch || child != ctx.Engine.CurrentBranch() {
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

		childLines := getUpstackInclusiveLines(printStackArgs{
			short:         args.short,
			reverse:       args.reverse,
			branchName:    child,
			indentLevel:   childIndent,
			steps:         childSteps,
			omitCurrentBranch: args.omitCurrentBranch,
			noStyleBranchName: args.noStyleBranchName,
			overallIndent: args.overallIndent,
		}, ctx)

		result = append(result, childLines...)
	}

	return result
}

func getUpstackInclusiveLines(args printStackArgs, ctx *context.Context) []string {
	outputDeep := [][]string{
		getUpstackExclusiveLines(args, ctx),
		getBranchLines(args, ctx),
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

func getDownstackExclusiveLines(args printStackArgs, ctx *context.Context) []string {
	if ctx.Engine.IsTrunk(args.branchName) {
		return []string{}
	}

	stack := ctx.Engine.GetRelativeStack(args.branchName, engine.Scope{
		RecursiveParents: true,
	})

	// Prepend trunk
	fullStack := append([]string{ctx.Engine.Trunk()}, stack...)

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
		branchLines := getBranchLines(printStackArgs{
			short:            args.short,
			reverse:          args.reverse,
			branchName:       branchName,
			indentLevel:      args.indentLevel,
			skipBranchingLine: true,
		}, ctx)
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

func getBranchLines(args printStackArgs, ctx *context.Context) []string {
	children := ctx.Engine.GetChildren(args.branchName)
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
		isCurrent := args.branchName == ctx.Engine.CurrentBranch()
		if isCurrent && !args.noStyleBranchName {
			line += "◉"
		} else {
			line += "◯"
		}
		line += "▸" + args.branchName

		// Add restack indicator
		if !args.noStyleBranchName && !ctx.Engine.IsBranchFixed(args.branchName) {
			line += " (needs restack)"
		}

		return []string{line}
	}

	// Full format
	var result []string

	// Branching line
	if !args.skipBranchingLine && numChildren >= 2 {
		result = append(result, getBranchingLine(numChildren, args.reverse, args.indentLevel))
	}

	// Branch info lines
	infoLines := getInfoLines(args, ctx)
	result = append(result, infoLines...)

	if args.reverse {
		for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
			result[i], result[j] = result[j], result[i]
		}
	}

	return result
}

func getBranchingLine(numChildren int, reverse bool, indentLevel int) string {
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

func getInfoLines(args printStackArgs, ctx *context.Context) []string {
	isCurrent := args.branchName == ctx.Engine.CurrentBranch()
	
	// Get branch info with colors
	branchName := args.branchName
	coloredBranchName := output.ColorBranchName(branchName, isCurrent)
	
	// Add restack indicator if needed
	if !ctx.Engine.IsBranchFixed(branchName) {
		coloredBranchName += " " + output.ColorNeedsRestack("(needs restack)")
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

func formatShortLines(lines []string, args printStackArgs, ctx *context.Context) []string {
	var result []string
	currentBranch := ctx.Engine.CurrentBranch()

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
			isCurrent := !args.noStyleBranchName && currentBranch != "" && branchName == currentBranch

			overallIndent := 0
			if args.overallIndent != nil {
				overallIndent = *args.overallIndent
			}

			formatted := output.FormatShortLine(line, circleIndex, arrowIndex, isCurrent, overallIndent)
			result = append(result, formatted)
		} else {
			result = append(result, line)
		}
	}

	return result
}

func getUntrackedBranchNames(ctx *context.Context) []string {
	var untracked []string
	for _, branchName := range ctx.Engine.AllBranchNames() {
		if !ctx.Engine.IsTrunk(branchName) && !ctx.Engine.IsBranchTracked(branchName) {
			untracked = append(untracked, branchName)
		}
	}
	return untracked
}

