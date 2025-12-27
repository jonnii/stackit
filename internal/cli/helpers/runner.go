package helpers

import (
	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/runtime"
)

// Run is a helper that provides a runtime context to a command's execution function
func Run(cmd *cobra.Command, fn func(ctx *runtime.Context) error) error {
	ctx, err := runtime.GetContext(cmd.Context())
	if err != nil {
		return err
	}
	return fn(ctx)
}
