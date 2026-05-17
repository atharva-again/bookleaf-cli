package cmd

import (
	"encoding/json"
	"fmt"

	"bookleaf-cli/internal/format"
	"bookleaf-cli/internal/mcp"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

var ticketsCmd = &cobra.Command{
	Use:     "ticket",
	Aliases: []string{"tickets"},
	Short:   "Manage support tickets",
}

func getMcpClient() *mcp.Client {
	if cfg.Auth == nil {
		format.PrintFatal("Not authenticated. Run 'bookleaf auth login' first.")
	}
	return mcp.NewClient(cfg.APIURL, cfg.Auth.AccessToken)
}

func requireAdmin() {
	if cfg.Auth == nil {
		format.PrintFatal("Not authenticated. Run 'bookleaf auth login' first.")
	}
	if cfg.Auth.Role != "admin" {
		format.PrintFatal("This command requires admin privileges. Your role: %s", cfg.Auth.Role)
	}
}

func output(result string) {
	outputTool("", result)
}

func outputTool(toolName string, result string) {
	if cfg.UseJSON {
		var parsed any
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			fmt.Println(result)
			return
		}
		format.PrintJSON(parsed)
		return
	}
	fmt.Print(format.FormatToolResponse(toolName, result))
}

var ticketsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List support tickets",
	RunE: func(_ *cobra.Command, _ []string) error {
		client := getMcpClient()

		var result string
		var toolName string
		var err error

		args := map[string]any{}
		if listStatus != "" {
			args["status"] = listStatus
		}
		if listPriority != "" {
			args["priority"] = listPriority
		}
		if listCategory != "" {
			args["category"] = listCategory
		}
		if listSearch != "" {
			args["search"] = listSearch
		}
		if listLimit > 0 {
			args["limit"] = listLimit
		}
		if listOffset > 0 {
			args["offset"] = listOffset
		}

		if cfg.Auth != nil && cfg.Auth.Role == "admin" {
			toolName = "list_tickets"
			if listAssignedTo != "" {
				args["assignedTo"] = listAssignedTo
			}
		} else {
			toolName = "list_my_tickets"
		}

		result, err = client.CallTool(toolName, args)

		if err != nil {
			return fmt.Errorf("list tickets: %w", err)
		}

		outputTool(toolName, result)
		return nil
	},
}

var listStatus string
var listPriority string
var listCategory string
var listAssignedTo string
var listSearch string
var listLimit int
var listOffset int

var ticketsViewCmd = &cobra.Command{
	Use:   "view <id>",
	Short: "View ticket details with messages",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := getMcpClient()

		result, err := client.CallTool("get_ticket", map[string]any{
			"id": args[0],
		})
		if err != nil {
			return fmt.Errorf("view ticket: %w", err)
		}

		outputTool("get_ticket", result)
		return nil
	},
}

var ticketsCreateCmd = &cobra.Command{
	Use:   "create [subject]",
	Short: "Create a new support ticket",
	Long: `Create a new support ticket.

With no arguments and --body, prompts interactively for subject,
description, and book selection when stdin is a terminal.

Alias: tickets`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		var subject, body, bookID, bookTitle string

		if len(args) > 0 {
			subject = args[0]
		} else if isInteractive() {
			if err := huh.NewInput().Title("Subject").Value(&subject).Run(); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("subject is required (provide as argument or run interactively)")
		}

		if cmd.Flags().Changed("body") {
			body = createBody
		} else if isInteractive() {
			if err := huh.NewText().Title("Description").ExternalEditor(true).Value(&body).Run(); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("--body is required")
		}

		if cmd.Flags().Changed("book-id") {
			bookID = createBookID
		} else if isInteractive() {
			client := getMcpClient()

			toolName := "list_my_books"
			if cfg.Auth != nil && cfg.Auth.Role == "admin" {
				toolName = "list_books"
			}

			result, err := client.CallTool(toolName, nil)
			if err == nil {
				var books []map[string]any
				if json.Unmarshal([]byte(result), &books) == nil && len(books) > 0 {
					opts := make([]huh.Option[string], 0, len(books)+1)
					opts = append(opts, huh.NewOption("(none)", ""))
					for _, b := range books {
						id, _ := b["id"].(string)
						title, _ := b["title"].(string)
						opts = append(opts, huh.NewOption(title, id))
					}
					if err := huh.NewSelect[string]().Title("Book").Options(opts...).Value(&bookID).Run(); err != nil {
						return err
					}
					for _, b := range books {
						if id, _ := b["id"].(string); id == bookID {
							bookTitle, _ = b["title"].(string)
							break
						}
					}
				}
			}
		}

		if isInteractive() {
			fmt.Println()
			fmt.Print("  \033[1mSummary\033[0m\n")
			fmt.Printf("    Subject:     %s\n", subject)
			fmt.Printf("    Description: %s\n", body)
			if bookTitle != "" {
				fmt.Printf("    Book:        %s\n", bookTitle)
			}
			fmt.Println()

			var confirmed bool
			if err := huh.NewConfirm().
				Title("Create this ticket?").
				Affirmative("Yes").
				Negative("No").
				Value(&confirmed).
				Run(); err != nil {
				return err
			}
			if !confirmed {
				format.PrintSuccess("Cancelled")
				return nil
			}
		}

		client := getMcpClient()
		mcpArgs := map[string]any{
			"subject":     subject,
			"description": body,
		}
		if bookID != "" {
			mcpArgs["bookId"] = bookID
		}

		result, err := client.CallTool("create_ticket", mcpArgs)
		if err != nil {
			return fmt.Errorf("create ticket: %w", err)
		}

		outputTool("create_ticket", result)
		return nil
	},
}

var createBody string
var createBookID string

var ticketsRespondCmd = &cobra.Command{
	Use:   "respond <id>",
	Short: "Respond to a ticket (admin only)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAdmin()

		body := respondBody
		if !cmd.Flags().Changed("body") && isInteractive() {
			if err := huh.NewText().Title("Response").ExternalEditor(true).Value(&body).Run(); err != nil {
				return err
			}
		} else if body == "" {
			return fmt.Errorf("--body is required")
		}

		client := getMcpClient()
		result, err := client.CallTool("respond_to_ticket", map[string]any{
			"ticketId":  args[0],
			"body":      body,
			"isAiDraft": respondAiDraft,
		})
		if err != nil {
			return fmt.Errorf("respond: %w", err)
		}

		outputTool("respond_to_ticket", result)
		return nil
	},
}

var respondBody string
var respondAiDraft bool

var ticketsNoteCmd = &cobra.Command{
	Use:   "note <id>",
	Short: "Add an internal note (admin only)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAdmin()

		body := noteBody
		if !cmd.Flags().Changed("body") && isInteractive() {
			if err := huh.NewText().Title("Note").ExternalEditor(true).Value(&body).Run(); err != nil {
				return err
			}
		} else if body == "" {
			return fmt.Errorf("--body is required")
		}

		client := getMcpClient()
		result, err := client.CallTool("add_note", map[string]any{
			"ticketId": args[0],
			"body":     body,
		})
		if err != nil {
			return fmt.Errorf("add note: %w", err)
		}

		outputTool("add_note", result)
		return nil
	},
}

var noteBody string

var ticketsUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a ticket (admin only)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		requireAdmin()

		status := updateStatus
		priority := updatePriority
		category := updateCategory
		assignedTo := updateAssignedTo

		if !cmd.Flags().Changed("status") && isInteractive() {
			if err := huh.NewSelect[string]().
				Title("Status").
				Options(
					huh.NewOption("Open", "open"),
					huh.NewOption("In Progress", "in_progress"),
					huh.NewOption("Resolved", "resolved"),
					huh.NewOption("Closed", "closed"),
				).
				Value(&status).
				Run(); err != nil {
				return err
			}
		}

		if !cmd.Flags().Changed("priority") && isInteractive() {
			if err := huh.NewSelect[string]().
				Title("Priority").
				Options(
					huh.NewOption("Critical", "critical"),
					huh.NewOption("High", "high"),
					huh.NewOption("Medium", "medium"),
					huh.NewOption("Low", "low"),
				).
				Value(&priority).
				Run(); err != nil {
				return err
			}
		}

		if !cmd.Flags().Changed("category") && isInteractive() {
			if err := huh.NewSelect[string]().
				Title("Category").
				Options(
					huh.NewOption("Royalty & Payments", "royalty_payments"),
					huh.NewOption("ISBN & Metadata", "isbn_metadata"),
					huh.NewOption("Printing & Quality", "printing_quality"),
					huh.NewOption("Distribution & Availability", "distribution_availability"),
					huh.NewOption("Production Updates", "book_status_production"),
					huh.NewOption("General", "general"),
				).
				Value(&category).
				Run(); err != nil {
				return err
			}
		}

		if !cmd.Flags().Changed("assigned-to") && isInteractive() {
			if err := huh.NewInput().
				Title("Assigned To").
				Placeholder("Admin user ID").
				Value(&assignedTo).
				Run(); err != nil {
				return err
			}
		}

		if status == "" && priority == "" && category == "" && assignedTo == "" {
			return fmt.Errorf("at least one field to update is required")
		}

		mcpArgs := map[string]any{
			"id": args[0],
		}
		if status != "" {
			mcpArgs["status"] = status
		}
		if priority != "" {
			mcpArgs["priority"] = priority
		}
		if category != "" {
			mcpArgs["category"] = category
		}
		if assignedTo != "" {
			mcpArgs["assignedTo"] = assignedTo
		}

		client := getMcpClient()
		result, err := client.CallTool("update_ticket", mcpArgs)
		if err != nil {
			return fmt.Errorf("update ticket: %w", err)
		}

		outputTool("update_ticket", result)
		return nil
	},
}

var updateStatus string
var updatePriority string
var updateCategory string
var updateAssignedTo string

var ticketsDraftCmd = &cobra.Command{
	Use:   "draft <id>",
	Short: "Generate an AI-drafted response (admin only)",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		requireAdmin()
		client := getMcpClient()

		result, err := client.CallTool("generate_ai_draft", map[string]any{
			"ticketId": args[0],
		})
		if err != nil {
			return fmt.Errorf("generate draft: %w", err)
		}

		outputTool("generate_ai_draft", result)
		return nil
	},
}

func init() {
	ticketsListCmd.Flags().StringVar(&listStatus, "status", "", "Filter by status (open, in_progress, resolved, closed)")
	ticketsListCmd.Flags().StringVar(&listPriority, "priority", "", "Filter by priority (critical, high, medium, low)")
	ticketsListCmd.Flags().StringVar(&listCategory, "category", "", "Filter by category")
	ticketsListCmd.Flags().StringVar(&listAssignedTo, "assigned-to", "", "Filter by assigned admin")
	ticketsListCmd.Flags().StringVar(&listSearch, "search", "", "Search in subject/description")
	ticketsListCmd.Flags().IntVar(&listLimit, "limit", 0, "Max results")
	ticketsListCmd.Flags().IntVar(&listOffset, "offset", 0, "Pagination offset")
	ticketsCmd.AddCommand(ticketsListCmd)

	ticketsViewCmd.Flags().String("id", "", "Ticket ID (positional)")
	ticketsCmd.AddCommand(ticketsViewCmd)

	ticketsCreateCmd.Flags().StringVar(&createBody, "body", "", "Ticket description")
	ticketsCreateCmd.Flags().StringVar(&createBookID, "book-id", "", "Related book ID")
	ticketsCmd.AddCommand(ticketsCreateCmd)

	ticketsRespondCmd.Flags().StringVar(&respondBody, "body", "", "Response text")
	ticketsRespondCmd.Flags().BoolVar(&respondAiDraft, "ai-draft", false, "Mark as AI-generated draft")
	ticketsCmd.AddCommand(ticketsRespondCmd)

	ticketsNoteCmd.Flags().StringVar(&noteBody, "body", "", "Note text")
	ticketsCmd.AddCommand(ticketsNoteCmd)

	ticketsUpdateCmd.Flags().StringVar(&updateStatus, "status", "", "New status")
	ticketsUpdateCmd.Flags().StringVar(&updatePriority, "priority", "", "New priority")
	ticketsUpdateCmd.Flags().StringVar(&updateCategory, "category", "", "New category")
	ticketsUpdateCmd.Flags().StringVar(&updateAssignedTo, "assigned-to", "", "Assign to admin ID")
	ticketsCmd.AddCommand(ticketsUpdateCmd)

	ticketsCmd.AddCommand(ticketsDraftCmd)
}
