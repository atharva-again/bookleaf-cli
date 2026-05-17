package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var booksCmd = &cobra.Command{
	Use:     "book",
	Aliases: []string{"books"},
	Short:   "List books",
}

var booksListCmd = &cobra.Command{
	Use:   "list",
	Short: "List books",
	RunE: func(_ *cobra.Command, _ []string) error {
		client := getMcpClient()

		var result string
		var toolName string
		var err error

		if cfg.Auth != nil && cfg.Auth.Role == "admin" {
			toolName = "list_books"
			args := map[string]any{}
			if bookListAuthorID != "" {
				args["authorId"] = bookListAuthorID
			}
			if bookListSearch != "" {
				args["search"] = bookListSearch
			}
			result, err = client.CallTool(toolName, args)
		} else {
			toolName = "list_my_books"
			result, err = client.CallTool(toolName, nil)
		}

		if err != nil {
			return fmt.Errorf("list books: %w", err)
		}

		outputTool(toolName, result)
		return nil
	},
}

var bookListAuthorID string
var bookListSearch string

func init() {
	booksListCmd.Flags().StringVar(&bookListAuthorID, "author-id", "", "Filter by author ID")
	booksListCmd.Flags().StringVar(&bookListSearch, "search", "", "Search by title or ISBN")
	booksCmd.AddCommand(booksListCmd)
}
