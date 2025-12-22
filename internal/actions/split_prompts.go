package actions

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/utils"
)

func promptBranchName(existingNames []string, originalBranchName string, branchNum int, eng engine.BranchReader) (string, error) {
	defaultName := originalBranchName
	if utils.ContainsString(existingNames, defaultName) {
		defaultName = originalBranchName + "_split"
		for utils.ContainsString(existingNames, defaultName) {
			defaultName += "_split"
		}
	}

	var branchName string
	prompt := &survey.Input{
		Message: fmt.Sprintf("Choose a name for branch %d", branchNum),
		Default: defaultName,
	}
	if err := survey.AskOne(prompt, &branchName); err != nil {
		return "", fmt.Errorf("canceled")
	}

	// Validate name - don't allow names already picked in this split session
	if utils.ContainsString(existingNames, branchName) {
		return "", fmt.Errorf("branch name %s is already used by another branch in this split", branchName)
	}

	// Allow reusing the original branch name being split (it will be replaced)
	// but don't allow other existing branch names
	if branchName != originalBranchName {
		allBranches := eng.AllBranchNames()
		if utils.ContainsString(allBranches, branchName) {
			return "", fmt.Errorf("branch name %s is already in use", branchName)
		}
	}

	return branchName, nil
}

func makeRange(n int) []int {
	result := make([]int, n)
	for i := 0; i < n; i++ {
		result[i] = i
	}
	return result
}
