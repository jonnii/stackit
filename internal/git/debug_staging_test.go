package git_test

import (
    "fmt"
    "os"
    "testing"

    "github.com/stretchr/testify/require"
    "stackit.dev/stackit/internal/git"
    "stackit.dev/stackit/testhelpers"
)

func TestDebugHasUnstagedChanges(t *testing.T) {
    scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
        return s.Repo.CreateChangeAndCommit("initial", "test")
    })

    // Create unstaged change
    err := scene.Repo.CreateChange("modified", "test", true)
    require.NoError(t, err)

    // Debug: check what directory we're in
    cwd, _ := os.Getwd()
    fmt.Printf("Current dir: %s\n", cwd)
    fmt.Printf("Scene dir: %s\n", scene.Dir)
    
    // Check if file exists
    files, _ := os.ReadDir(".")
    fmt.Printf("Files in current dir: ")
    for _, f := range files {
        fmt.Printf("%s ", f.Name())
    }
    fmt.Println()
    
    // Check git status output directly
    output, err := git.RunGitCommand("status", "--porcelain")
    fmt.Printf("Git status output: %q\n", output)
    require.NoError(t, err)

    hasUnstaged, err := git.HasUnstagedChanges()
    fmt.Printf("HasUnstagedChanges result: %v\n", hasUnstaged)
    require.NoError(t, err)
    require.True(t, hasUnstaged)
}
