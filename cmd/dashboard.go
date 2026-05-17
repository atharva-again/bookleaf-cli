package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Show your author dashboard",
	RunE: func(_ *cobra.Command, _ []string) error {
		client := getMcpClient()

		result, err := client.CallTool("get_my_dashboard", nil)
		if err != nil {
			return fmt.Errorf("dashboard: %w", err)
		}

		outputTool("get_my_dashboard", result)
		return nil
	},
}
