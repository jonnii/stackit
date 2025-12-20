package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSanitizeBranchName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name passes through",
			input:    "feature",
			expected: "feature",
		},
		{
			name:     "spaces replaced with hyphens",
			input:    "my feature branch",
			expected: "my-feature-branch",
		},
		{
			name:     "special characters replaced",
			input:    "feature!@#$%^&*()",
			expected: "feature",
		},
		{
			name:     "underscores preserved",
			input:    "my_feature_branch",
			expected: "my_feature_branch",
		},
		{
			name:     "slashes preserved",
			input:    "feature/my-branch",
			expected: "feature/my-branch",
		},
		{
			name:     "dots preserved",
			input:    "feature.v1.0",
			expected: "feature.v1.0",
		},
		{
			name:     "trailing dots removed",
			input:    "feature...",
			expected: "feature",
		},
		{
			name:     "trailing slashes removed",
			input:    "feature///",
			expected: "feature",
		},
		{
			name:     "multiple consecutive hyphens collapsed",
			input:    "my---feature---branch",
			expected: "my-feature-branch",
		},
		{
			name:     "leading hyphens trimmed",
			input:    "---feature",
			expected: "feature",
		},
		{
			name:     "trailing hyphens trimmed",
			input:    "feature---",
			expected: "feature",
		},
		{
			name:     "mixed invalid characters",
			input:    "feat: add new feature!",
			expected: "feat-add-new-feature",
		},
		{
			name:     "numbers preserved",
			input:    "feature123",
			expected: "feature123",
		},
		{
			name:     "mixed case preserved",
			input:    "MyFeatureBranch",
			expected: "MyFeatureBranch",
		},
		{
			name:     "empty string returns empty",
			input:    "",
			expected: "",
		},
		{
			name:     "only special chars returns empty",
			input:    "!@#$%",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := SanitizeBranchName(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeBranchName_MaxLength(t *testing.T) {
	t.Parallel()

	// Create a string longer than MaxBranchNameByteLength
	longName := strings.Repeat("a", MaxBranchNameByteLength+50)

	result := SanitizeBranchName(longName)

	require.LessOrEqual(t, len(result), MaxBranchNameByteLength)
	require.Equal(t, MaxBranchNameByteLength, len(result))
}

func TestSanitizeBranchName_MaxLengthTrimsTrailingHyphen(t *testing.T) {
	t.Parallel()

	// Create a string that when truncated would end with a hyphen
	// MaxBranchNameByteLength is 234, so we create a string where position 234 is a hyphen
	longName := strings.Repeat("a", MaxBranchNameByteLength-1) + "-" + strings.Repeat("b", 50)

	result := SanitizeBranchName(longName)

	require.LessOrEqual(t, len(result), MaxBranchNameByteLength)
	require.False(t, strings.HasSuffix(result, "-"), "result should not end with hyphen")
}

func TestGenerateBranchNameFromMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "simple message",
			message:  "Add new feature",
			expected: "Add-new-feature",
		},
		{
			name:     "message with conventional commit prefix feat",
			message:  "feat: add new feature",
			expected: "add-new-feature",
		},
		{
			name:     "message with conventional commit prefix fix",
			message:  "fix: resolve bug",
			expected: "resolve-bug",
		},
		{
			name:     "message with conventional commit prefix chore",
			message:  "chore: update dependencies",
			expected: "update-dependencies",
		},
		{
			name:     "message with conventional commit prefix docs",
			message:  "docs: update readme",
			expected: "update-readme",
		},
		{
			name:     "message with conventional commit prefix style",
			message:  "style: format code",
			expected: "format-code",
		},
		{
			name:     "message with conventional commit prefix refactor",
			message:  "refactor: improve structure",
			expected: "improve-structure",
		},
		{
			name:     "message with conventional commit prefix perf",
			message:  "perf: optimize query",
			expected: "optimize-query",
		},
		{
			name:     "message with conventional commit prefix test",
			message:  "test: add unit tests",
			expected: "add-unit-tests",
		},
		{
			name:     "message with conventional commit prefix build",
			message:  "build: update config",
			expected: "update-config",
		},
		{
			name:     "message with conventional commit prefix ci",
			message:  "ci: update pipeline",
			expected: "update-pipeline",
		},
		{
			name:     "multiline message uses first line only",
			message:  "First line\nSecond line\nThird line",
			expected: "First-line",
		},
		{
			name:     "message with special characters",
			message:  "Add feature! (for users)",
			expected: "Add-feature-for-users",
		},
		{
			name:     "empty message returns empty",
			message:  "",
			expected: "",
		},
		{
			name:     "whitespace only message",
			message:  "   ",
			expected: "",
		},
		{
			name:     "message with leading/trailing whitespace",
			message:  "  Add feature  ",
			expected: "Add-feature",
		},
		{
			name:     "prefix without space after colon",
			message:  "feat:add feature",
			expected: "add-feature",
		},
		{
			name:     "long message gets truncated",
			message:  "feat: add a very long feature description that should be truncated to fit in branch names",
			expected: "add-a-very-long-feature-description-that-should",
		},
		{
			name:     "message with scope prefix",
			message:  "fix(ai): enhance error reporting",
			expected: "enhance-error-reporting",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := GenerateBranchNameFromMessage(tt.message)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestProcessBranchNamePattern(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		pattern  string
		username string
		date     string
		message  string
		expected string
	}{
		{
			name:     "default pattern with all placeholders",
			pattern:  "{username}/{date}/{message}",
			username: "jonnii",
			date:     "20251218123456",
			message:  "Add new feature",
			expected: "jonnii/20251218123456/Add-new-feature",
		},
		{
			name:     "pattern with only message",
			pattern:  "{message}",
			username: "jonnii",
			date:     "20251218123456",
			message:  "Add new feature",
			expected: "Add-new-feature",
		},
		{
			name:     "pattern with username and message",
			pattern:  "{username}/{message}",
			username: "jonnii",
			date:     "20251218123456",
			message:  "Add new feature",
			expected: "jonnii/Add-new-feature",
		},
		{
			name:     "pattern with date and message",
			pattern:  "{date}/{message}",
			username: "jonnii",
			date:     "20251218123456",
			message:  "Add new feature",
			expected: "20251218123456/Add-new-feature",
		},
		{
			name:     "empty pattern uses message only",
			pattern:  "",
			username: "jonnii",
			date:     "20251218123456",
			message:  "Add new feature",
			expected: "Add-new-feature",
		},
		{
			name:     "pattern with special characters in username",
			pattern:  "{username}/{date}/{message}",
			username: "john doe",
			date:     "20251218123456",
			message:  "Add feature",
			expected: "john-doe/20251218123456/Add-feature",
		},
		{
			name:     "pattern with conventional commit prefix in message",
			pattern:  "{username}/{date}/{message}",
			username: "jonnii",
			date:     "20251218123456",
			message:  "feat: add new feature",
			expected: "jonnii/20251218123456/add-new-feature",
		},
		{
			name:     "pattern with multiple slashes",
			pattern:  "{username}/dev/{date}/{message}",
			username: "jonnii",
			date:     "20251218123456",
			message:  "Add feature",
			expected: "jonnii/dev/20251218123456/Add-feature",
		},
		{
			name:     "pattern with empty username",
			pattern:  "{username}/{date}/{message}",
			username: "",
			date:     "20251218123456",
			message:  "Add feature",
			expected: "/20251218123456/Add-feature",
		},
		{
			name:     "pattern with empty message",
			pattern:  "{username}/{date}/{message}",
			username: "jonnii",
			date:     "20251218123456",
			message:  "",
			expected: "jonnii/20251218123456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := ProcessBranchNamePattern(tt.pattern, tt.username, tt.date, tt.message)
			require.Equal(t, tt.expected, result)
		})
	}
}
