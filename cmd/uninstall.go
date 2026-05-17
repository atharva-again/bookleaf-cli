package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"bookleaf-cli/internal/config"
	"bookleaf-cli/internal/format"

	"github.com/spf13/cobra"
)

var uninstallForce bool

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove the bookleaf binary and configuration",
	Long: `Removes the bookleaf binary and deletes the configuration directory (~/.config/bookleaf/).

Use --force to skip the confirmation prompt.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		binaryPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("find binary path: %w", err)
		}
		binaryPath, err = filepath.EvalSymlinks(binaryPath)
		if err != nil {
			return fmt.Errorf("resolve binary path: %w", err)
		}

		configPath, err := config.Path()
		if err != nil {
			return fmt.Errorf("find config path: %w", err)
		}
		configDir := filepath.Dir(configPath)

		if !uninstallForce {
			fmt.Println("This will:")
			fmt.Printf("  1. Delete the binary: %s\n", binaryPath)
			fmt.Printf("  2. Delete the config directory: %s\n", configDir)
			fmt.Print("\nAre you sure? [y/N]: ")
			var response string
			if _, err := fmt.Scanln(&response); err != nil {
				return nil
			}
			if response != "y" && response != "Y" {
				fmt.Println("Uninstall cancelled.")
				return nil
			}
		}

		if _, err := os.Stat(configDir); err == nil {
			if err := os.RemoveAll(configDir); err != nil {
				return fmt.Errorf("delete config directory: %w", err)
			}
		}

		if err := os.Remove(binaryPath); err != nil {
			if os.IsPermission(err) {
				fmt.Println("Need sudo to remove the binary...")
				cmd := exec.Command("sudo", "rm", binaryPath)
				cmd.Stdin = os.Stdin
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				if err := cmd.Run(); err != nil {
					return fmt.Errorf("sudo rm binary: %w", err)
				}
			} else {
				return fmt.Errorf("delete binary: %w", err)
			}
		}

		format.PrintSuccess("BookLeaf CLI uninstalled")
		return nil
	},
}

func init() {
	uninstallCmd.Flags().BoolVarP(&uninstallForce, "force", "f", false, "Skip confirmation prompt")
	rootCmd.AddCommand(uninstallCmd)
}
