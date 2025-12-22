package actions

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"

	"stackit.dev/stackit/internal/engine"
)

func promptBranchName(existingNames []string, originalBranchName string, branchNum int, eng engine.BranchReader) (string, error) {
	defaultName := originalBranchName
	if containsString(existingNames, defaultName) {
		defaultName = originalBranchName + "_split"
		for containsString(existingNames, defaultName) {
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
	if containsString(existingNames, branchName) {
		return "", fmt.Errorf("branch name %s is already used by another branch in this split", branchName)
	}

	// Allow reusing the original branch name being split (it will be replaced)
	// but don't allow other existing branch names
	if branchName != originalBranchName {
		allBranches := eng.AllBranchNames()
		if containsString(allBranches, branchName) {
			return "", fmt.Errorf("branch name %s is already in use", branchName)
		}
	}

	return branchName, nil
}

func pluralize(word string, count int) string {
	if count == 1 {
		return word
	}
	return word + "ren" // "child" -> "children"
}

func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func makeRange(n int) []int {
	result := make([]int, n)
	for i := 0; i < n; i++ {
		result[i] = i
	}
	return result
}
