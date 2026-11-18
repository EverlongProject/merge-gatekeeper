package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

// These variables will be set by command line flags.
var (
	ghToken string
)

func Run(version string, args ...string) error {
	cmd := &cobra.Command{
		Use:     "merge-gatekeeper",
		Short:   "Get more refined merge control",
		Version: version,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if ghToken == "" {
				ghToken = os.Getenv("GITHUB_TOKEN")
			}
			if ghToken == "" {
				return fmt.Errorf("github token is required: set --token or the GITHUB_TOKEN environment variable")
			}
			fmt.Printf("Using GitHub token %s\n", ghToken)
			return nil
		},
	}
	cmd.PersistentFlags().StringVarP(&ghToken, "token", "t", "", "set github token (defaults to the GITHUB_TOKEN environment variable)")

	cmd.AddCommand(validateCmd())

	ctx, cancel := signal.NotifyContext(context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer cancel()

	if err := cmd.ExecuteContext(ctx); err != nil {
		return err
	}
	return nil
}
