package format

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
	colorBold   = "\033[1m"
)

// PrintJSON outputs data as formatted JSON.
func PrintJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

// PrintTable outputs rows as an aligned table with a header row.
func PrintTable(header []string, rows [][]string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)

	for i, h := range header {
		if i > 0 {
			fmt.Fprint(w, "\t")
		}
		fmt.Fprintf(w, "%s%s%s%s", colorBold, colorBlue, h, colorReset)
	}
	fmt.Fprintln(w)

	for i := range header {
		if i > 0 {
			fmt.Fprint(w, "\t")
		}
		fmt.Fprint(w, strings.Repeat("-", len(header[i])))
	}
	fmt.Fprintln(w)

	for _, row := range rows {
		for i, cell := range row {
			if i > 0 {
				fmt.Fprint(w, "\t")
			}
		if len(cell) > 60 {
				cell = cell[:57] + "..."
			}
			fmt.Fprint(w, cell)
		}
		fmt.Fprintln(w)
	}

	_ = w.Flush()
}

// PrintKV outputs key-value pairs, one per line.
func PrintKV(pairs ...string) {
	if len(pairs)%2 != 0 {
		return
	}
	for i := 0; i < len(pairs); i += 2 {
		fmt.Printf("  %s%s%s  %s\n", colorCyan, pairs[i], colorReset, pairs[i+1])
	}
}

// PrintSuccess prints a green success message.
func PrintSuccess(msg string) {
	fmt.Fprintf(os.Stdout, "%s✓%s %s\n", colorGreen, colorReset, msg)
}

// PrintError prints a red error to stderr.
func PrintError(msg string) {
	fmt.Fprintf(os.Stderr, "%sError:%s %s\n", colorRed, colorReset, msg)
	os.Exit(1)
}

// PrintFatal prints a red error and exits with code 1.
func PrintFatal(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "%sError:%s %s\n", colorRed, colorReset, msg)
	os.Exit(1)
}

// StatusIcon returns a colored icon for a ticket status.
func StatusIcon(status string) string {
	switch strings.ToLower(status) {
	case "open":
		return fmt.Sprintf("%s●%s open", colorGreen, colorReset)
	case "in_progress":
		return fmt.Sprintf("%s◐%s in_progress", colorYellow, colorReset)
	case "resolved":
		return fmt.Sprintf("%s○%s resolved", colorBlue, colorReset)
	case "closed":
		return fmt.Sprintf("%s○%s closed", colorGray, colorReset)
	default:
		return status
	}
}

// PriorityLabel returns a colored priority label.
func PriorityLabel(p string) string {
	switch strings.ToLower(p) {
	case "critical":
		return fmt.Sprintf("%scritical%s", colorRed, colorReset)
	case "high":
		return fmt.Sprintf("%shigh%s", colorYellow, colorReset)
	case "medium":
		return fmt.Sprintf("%smedium%s", colorBlue, colorReset)
	case "low":
		return fmt.Sprintf("%slow%s", colorGray, colorReset)
	default:
		return p
	}
}
