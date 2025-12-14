package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/context"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
)

// newSubmitCmd creates the submit command
func newSubmitCmd() *cobra.Command {
	var (
		branch              string
		stack               bool
		force               bool
		dryRun              bool
		confirm             bool
		updateOnly          bool
		always              bool
		restack             bool
		draft               bool
		publish             bool
		edit                bool
		editTitle           bool
		editDescription     bool
		noEdit              bool
		noEditTitle         bool
		noEditDescription   bool
		reviewers           string
		teamReviewers       string
		mergeWhenReady      bool
		rerequestReview     bool
		view                bool
		web                 bool
		comment             string
		targetTrunk         string
		ignoreOutOfSyncTrunk bool
		cli                 bool
	)

	cmd := &cobra.Command{
		Use:   "submit",
		Short: "Idempotently force push all branches in the current stack and create/update pull requests",
		Long: `Idempotently force push all branches in the current stack from trunk to the current branch to GitHub,
creating or updating distinct pull requests for each. Validates that branches are properly restacked before submitting,
and fails if there are conflicts. Blocks force pushes to branches that overwrite branches that have changed since
you last submitted or got them. Opens an interactive prompt that allows you to input pull request metadata.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Initialize git repository
			if err := git.InitDefaultRepo(); err != nil {
				return fmt.Errorf("not a git repository: %w", err)
			}

			// Get repo root
			repoRoot, err := git.GetRepoRoot()
			if err != nil {
				return fmt.Errorf("failed to get repo root: %w", err)
			}

			// Check if initialized
			if !config.IsInitialized(repoRoot) {
				return fmt.Errorf("stackit not initialized. Run 'stackit init' first")
			}

			// Create engine
			eng, err := engine.NewEngine(repoRoot)
			if err != nil {
				return fmt.Errorf("failed to create engine: %w", err)
			}

			// Create context
			ctx := context.NewContext(eng)

			// Run submit action
			return actions.SubmitAction(actions.SubmitOptions{
				Branch:              branch,
				Stack:               stack,
				Force:               force,
				DryRun:              dryRun,
				Confirm:             confirm,
				UpdateOnly:          updateOnly,
				Always:              always,
				Restack:             restack,
				Draft:               draft,
				Publish:             publish,
				Edit:                edit,
				EditTitle:           editTitle,
				EditDescription:     editDescription,
				NoEdit:              noEdit,
				NoEditTitle:         noEditTitle,
				NoEditDescription:   noEditDescription,
				Reviewers:           reviewers,
				TeamReviewers:       teamReviewers,
				MergeWhenReady:      mergeWhenReady,
				RerequestReview:     rerequestReview,
				View:                view,
				Web:                 web,
				Comment:             comment,
				TargetTrunk:         targetTrunk,
				IgnoreOutOfSyncTrunk: ignoreOutOfSyncTrunk,
				Engine:              eng,
				Splog:               ctx.Splog,
			})
		},
	}

	// Add flags
	cmd.Flags().StringVar(&branch, "branch", "", "Which branch to run this command from. Defaults to the current branch.")
	cmd.Flags().BoolVarP(&stack, "stack", "s", false, "Submit descendants of the current branch in addition to its ancestors.")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force push: overwrites the remote branch with your local branch. Otherwise defaults to --force-with-lease.")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Reports the PRs that would be submitted and terminates. No branches are restacked or pushed and no PRs are opened or updated.")
	cmd.Flags().BoolVarP(&confirm, "confirm", "c", false, "Reports the PRs that would be submitted and asks for confirmation before pushing branches and opening/updating PRs.")
	cmd.Flags().BoolVarP(&updateOnly, "update-only", "u", false, "Only push branches and update PRs for branches that already have PRs open.")
	cmd.Flags().BoolVar(&always, "always", false, "Always push updates, even if the branch has not changed.")
	cmd.Flags().BoolVar(&restack, "restack", false, "Restack branches before submitting.")
	cmd.Flags().BoolVarP(&draft, "draft", "d", false, "If set, all new PRs will be created in draft mode.")
	cmd.Flags().BoolVarP(&publish, "publish", "p", false, "If set, publishes all PRs being submitted.")
	cmd.Flags().BoolVarP(&edit, "edit", "e", false, "Input metadata for all PRs interactively.")
	cmd.Flags().BoolVar(&editTitle, "edit-title", false, "Input the PR title interactively.")
	cmd.Flags().BoolVar(&editDescription, "edit-description", false, "Input the PR description interactively.")
	cmd.Flags().BoolVarP(&noEdit, "no-edit", "n", false, "Don't edit any PR fields inline.")
	cmd.Flags().BoolVar(&noEditTitle, "no-edit-title", false, "Don't prompt for the PR title.")
	cmd.Flags().BoolVar(&noEditDescription, "no-edit-description", false, "Don't prompt for the PR description.")
	cmd.Flags().StringVar(&reviewers, "reviewers", "", "If set without an argument, prompt to manually set reviewers. Alternatively, accepts a comma separated string of reviewers.")
	cmd.Flags().StringVar(&teamReviewers, "team-reviewers", "", "Comma separated list of team slugs.")
	cmd.Flags().BoolVar(&mergeWhenReady, "merge-when-ready", false, "If set, marks all PRs being submitted as merge when ready.")
	cmd.Flags().BoolVar(&rerequestReview, "rerequest-review", false, "Rerequest review from current reviewers.")
	cmd.Flags().BoolVarP(&view, "view", "v", false, "Open the PR in your browser after submitting.")
	cmd.Flags().BoolVarP(&web, "web", "w", false, "Open a web browser to edit PR metadata.")
	cmd.Flags().StringVar(&comment, "comment", "", "Add a comment on the PR with the given message.")
	cmd.Flags().StringVarP(&targetTrunk, "target-trunk", "t", "", "Which trunk to open PRs against on remote.")
	cmd.Flags().BoolVar(&ignoreOutOfSyncTrunk, "ignore-out-of-sync-trunk", false, "Perform the submit operation even if the trunk branch is out of sync with its upstream branch.")
	cmd.Flags().BoolVar(&cli, "cli", false, "Edit PR metadata via the CLI instead of on web.")

	// Add alias - ss maps to submit --stack
	// We'll handle this by creating a separate command or using PersistentPreRun
	// For now, users can use "stackit submit --stack" or we can add a separate command

	return cmd
}
