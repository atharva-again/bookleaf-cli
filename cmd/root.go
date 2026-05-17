package cmd

import (
	"fmt"
	"os"

	"bookleaf-cli/internal/config"
	"bookleaf-cli/internal/format"

	"github.com/spf13/cobra"
)

// Set during build via ldflags (see .goreleaser.yaml)
var rootVersion = "dev"

var (
	cfg     *config.Config
	useJSON bool
)

var rootCmd = &cobra.Command{
	Use:   "bookleaf",
	Short: "BookLeaf CLI - Manage the BookLeaf support portal from your terminal",
	Long: `A command-line tool for interacting with the BookLeaf Publishing support portal.`,
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		if cmd.Name() == "help" || cmd.Name() == "completion" {
			return nil
		}

		var err error
		cfg, err = config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		if apiURLFlag != "" {
			cfg.APIURL = apiURLFlag
		}

		cfg.UseJSON = useJSON
		return nil
	},
}

var apiURLFlag string

func Execute() {
	rootCmd.Version = rootVersion
	if err := rootCmd.Execute(); err != nil {
		format.PrintError(err.Error())
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&useJSON, "json", false, "Output as JSON instead of formatted text")
	rootCmd.PersistentFlags().StringVar(&apiURLFlag, "api-url", "", "BookLeaf API URL (overrides config and env)")
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(ticketsCmd)
	rootCmd.AddCommand(booksCmd)
	rootCmd.AddCommand(dashboardCmd)
	rootCmd.AddCommand(whoamiCmd)
}

func init() {
	loaded, err := config.Load()
	if err != nil || loaded.Auth == nil || loaded.Auth.Role == "" {
		return
	}
	switch loaded.Auth.Role {
	case "admin":
		// Dashboard calls get_my_dashboard which is author-only (queries by authorId)
		dashboardCmd.Hidden = true
	case "author":
		ticketsRespondCmd.Hidden = true
		ticketsNoteCmd.Hidden = true
		ticketsUpdateCmd.Hidden = true
		ticketsDraftCmd.Hidden = true
	}
}
