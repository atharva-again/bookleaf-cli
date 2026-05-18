package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"bookleaf-cli/internal/config"
	"bookleaf-cli/internal/format"
	"bookleaf-cli/internal/mcp"

	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate the CLI with your BookLeaf account",
	Long: `Authenticate using OAuth Device Authorization flow.

A code will be displayed. Press Enter to copy it and open your browser,
then approve access on the web page. The CLI will automatically receive
a token once you authorize.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		client := mcp.NewDeviceAuthClient(cfg.APIURL)

		deviceResp, err := client.RequestDeviceCode()
		if err != nil {
			return fmt.Errorf("request device code: %w", err)
		}

		fmt.Println()
		fmt.Printf("  \033[1m\033[36m%s\033[0m\n", deviceResp.UserCode)
		fmt.Println()
		fmt.Printf("  Open this link in your browser:\n")
		fmt.Printf("  \033[34m%s\033[0m\n", deviceResp.VerificationURI)
		fmt.Println()
		fmt.Print("  Press \033[1mEnter\033[0m to copy the code and open the browser... ")
		bufio.NewReader(os.Stdin).ReadString('\n')

		copyToClipboard(deviceResp.UserCode)
		if err := openBrowser(deviceResp.VerificationURI); err != nil {
			fmt.Println()
			fmt.Printf("  Could not open browser automatically: %v\n", err)
			fmt.Printf("  Open this link manually: \033[34m%s\033[0m\n", deviceResp.VerificationURI)
		}

		fmt.Println()

		fmt.Print("  Waiting for authorization")

		tokenResp, err := client.PollForToken(deviceResp.DeviceCode, deviceResp.Interval)
		fmt.Println()

		if err != nil {
			return fmt.Errorf("poll for token: %w", err)
		}

		payload, err := mcp.DecodeToken(tokenResp.AccessToken)
		if err != nil {
			return fmt.Errorf("decode token: %w", err)
		}

		cfg.Auth = &config.Auth{
			AccessToken: tokenResp.AccessToken,
			UserID:      payload.UserID,
			Role:        payload.Role,
		}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		fmt.Println()
		format.PrintSuccess("Authenticated successfully")
		path, _ := config.Path()
		fmt.Printf("  Token stored in %s\n", path)
		fmt.Printf("  Role: %s\n", payload.Role)

		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear stored authentication",
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := config.ClearAuth(); err != nil {
			return fmt.Errorf("clear auth: %w", err)
		}
		format.PrintSuccess("Logged out")
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	RunE: func(_ *cobra.Command, _ []string) error {
		if cfg.Auth == nil || cfg.Auth.AccessToken == "" {
			fmt.Println("Not authenticated")
			fmt.Println("Run 'bookleaf auth login' to authenticate")
			return nil
		}

		if cfg.UseJSON {
			format.PrintJSON(cfg.Auth)
			return nil
		}

		fmt.Println("Authenticated")
		format.PrintKV(
			"Token:", cfg.Auth.AccessToken[:20]+"...",
			"Role:", cfg.Auth.Role,
		)
		if cfg.Auth.UserID != "" {
			format.PrintKV("User ID:", cfg.Auth.UserID)
		}
		if cfg.Auth.Email != "" {
			format.PrintKV("Email:", cfg.Auth.Email)
		}
		return nil
	},
}

func init() {
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)
}

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Display the current authenticated user",
	RunE: func(_ *cobra.Command, _ []string) error {
		if cfg.Auth == nil || cfg.Auth.AccessToken == "" {
			fmt.Println("Not authenticated")
			fmt.Println("Run 'bookleaf auth login' to authenticate")
			return nil
		}

		if cfg.UseJSON {
			format.PrintJSON(cfg.Auth)
			return nil
		}

		fmt.Printf("  Role:  %s\n", cfg.Auth.Role)
		if cfg.Auth.Email != "" {
			fmt.Printf("  Email: %s\n", cfg.Auth.Email)
		}
		if cfg.Auth.UserID != "" {
			fmt.Printf("  ID:    %s\n", cfg.Auth.UserID)
		}
		return nil
	},
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "windows":
		return exec.Command("cmd", "/c", "start", url).Start()
	default:
		if err := exec.Command("xdg-open", url).Start(); err == nil {
			return nil
		}
		return exec.Command("sensible-browser", url).Start()
	}
}

func copyToClipboard(text string) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "windows":
		cmd = exec.Command("clip")
	default:
		for _, prog := range []string{"wl-copy", "xclip", "xsel"} {
			if p, err := exec.LookPath(prog); err == nil {
				cmd = exec.Command(p)
				break
			}
		}
	}

	if cmd == nil {
		return
	}

	w, err := cmd.StdinPipe()
	if err != nil {
		return
	}
	if err := cmd.Start(); err != nil {
		return
	}
	w.Write([]byte(text))
	w.Close()
	cmd.Wait()
}
