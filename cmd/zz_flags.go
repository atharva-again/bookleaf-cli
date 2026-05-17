package cmd

import "bookleaf-cli/internal/config"

func init() {
	loaded, err := config.Load()
	if err != nil || loaded.Auth == nil || loaded.Auth.Role != "author" {
		return
	}

	// Authors can't use admin filter flags -- hide them from help
	for _, name := range []string{"status", "priority", "category", "assigned-to", "search", "limit", "offset"} {
		if f := ticketsListCmd.Flags().Lookup(name); f != nil {
			f.Hidden = true
		}
	}

	for _, name := range []string{"author-id", "search"} {
		if f := booksListCmd.Flags().Lookup(name); f != nil {
			f.Hidden = true
		}
	}
}
