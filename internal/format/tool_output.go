package format

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
)

func FormatToolResponse(toolName string, jsonStr string) string {
	var raw any
	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		return jsonStr
	}

	switch toolName {
	case "list_my_books", "list_books":
		return formatBookList(raw)
	case "list_my_tickets", "list_tickets":
		return formatTicketList(raw)
	case "get_ticket":
		return formatTicketDetail(raw)
	case "get_my_dashboard":
		return formatDashboard(raw)
	default:
		return formatDefault(raw)
	}
}

func newTable() table.Writer {
	tw := table.NewWriter()
	tw.SetStyle(table.StyleDefault)
	tw.Style().Options.SeparateHeader = true
	return tw
}

func formatBookList(raw any) string {
	arr, ok := raw.([]any)
	if !ok {
		return formatDefault(raw)
	}
	if len(arr) == 0 {
		return "  No books found.\n"
	}

	tw := newTable()
	tw.AppendHeader(table.Row{"Title", "ISBN", "Status", "Sold", "Earned", "Pending"})

	for _, item := range arr {
		book, ok := item.(map[string]any)
		if !ok {
			continue
		}
		title := truncate(str(book["title"]), 40)
		isbn := truncate(str(book["isbn"]), 18)
		status := str(book["status"])
		sold := fmt.Sprintf("%v", book["totalCopiesSold"])
		earned := fmt.Sprintf("Rs.%v", book["totalRoyaltyEarned"])
		pending := fmt.Sprintf("Rs.%v", book["royaltyPending"])
		tw.AppendRow(table.Row{title, isbn, status, sold, earned, pending})
	}

	return "\n" + tw.Render() + "\n"
}

func formatTicketList(raw any) string {
	arr, ok := raw.([]any)
	if !ok {
		return formatDefault(raw)
	}
	if len(arr) == 0 {
		return "  No tickets found.\n"
	}

	tw := newTable()
	tw.AppendHeader(table.Row{"#", "Subject", "Author", "Book", "Status", "Priority", "Category", "Age"})

	for _, item := range arr {
		t, ok := item.(map[string]any)
		if !ok {
			continue
		}
		num := "#-"
		if tn := t["ticketNumber"]; tn != nil {
			num = fmt.Sprintf("#%v", tn)
		}
		subject := truncate(str(t["subject"]), 50)
		author := truncate(str(t["authorName"]), 24)
		book := truncate(str(t["bookTitle"]), 30)
		if book == "" {
			book = "-"
		}
		status := StatusIcon(str(t["status"]))
		priority := PriorityLabel(str(t["priority"]))
		category := str(t["category"])
		age := formatAge(str(t["createdAt"]))
		tw.AppendRow(table.Row{num, subject, author, book, status, priority, category, age})
	}

	return "\n" + tw.Render() + "\n"
}

func formatAge(createdAt string) string {
	t, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return ""
	}
	hours := int(time.Since(t).Hours())
	if hours < 1 {
		minutes := int(time.Since(t).Minutes())
		return fmt.Sprintf("%dm", minutes)
	}
	if hours < 24 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf(">%dd", hours/24)
}

func formatTicketDetail(raw any) string {
	obj, ok := raw.(map[string]any)
	if !ok {
		return formatDefault(raw)
	}

	ticket, ok := obj["ticket"].(map[string]any)
	if !ok {
		return formatDefault(raw)
	}

	var b strings.Builder

	b.WriteString(fmt.Sprintf("\n  \033[1m\033[36m#%v - %s\033[0m\n", ticket["ticketNumber"], str(ticket["subject"])))
	b.WriteString(fmt.Sprintf("  %s  %s\n", StatusIcon(str(ticket["status"])), PriorityLabel(str(ticket["priority"]))))
	b.WriteString(fmt.Sprintf("  Category: %s\n", str(ticket["category"])))
	if desc, ok := ticket["description"].(string); ok && desc != "" {
		b.WriteString(fmt.Sprintf("\n  \033[33mDescription:\033[0m\n  %s\n", desc))
	}

	messages, _ := obj["messages"].([]any)
	if len(messages) > 0 {
		b.WriteString(fmt.Sprintf("\n  \033[1mMessages (%d):\033[0m\n", len(messages)))
		for _, msg := range messages {
			m := msg.(map[string]any)
			sender := str(m["senderName"])
			role := str(m["senderRole"])
			ts := formatTime(str(m["createdAt"]))
			draft := ""
			if bval, ok := m["isAiDraft"].(bool); ok && bval {
				draft = " \033[90m[AI draft]\033[0m"
			}
			body := str(m["body"])
			b.WriteString(fmt.Sprintf("\n  \033[36m%s\033[0m (\033[90m%s\033[0m) %s%s\n", sender, role, ts, draft))
			b.WriteString(fmt.Sprintf("  %s\n", body))
		}
	}

	notes, _ := obj["notes"].([]any)
	if len(notes) > 0 {
		b.WriteString(fmt.Sprintf("\n  \033[1m\033[33mInternal Notes (%d):\033[0m\n", len(notes)))
		for _, note := range notes {
			n := note.(map[string]any)
			admin := str(n["adminName"])
			ts := formatTime(str(n["createdAt"]))
			body := str(n["body"])
			b.WriteString(fmt.Sprintf("\n  \033[33m%s\033[0m (\033[90m%s\033[0m)\n", admin, ts))
			b.WriteString(fmt.Sprintf("  %s\n", body))
		}
	}

	b.WriteString("\n")
	return b.String()
}

func formatDashboard(raw any) string {
	obj, ok := raw.(map[string]any)
	if !ok {
		return formatDefault(raw)
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("\n  Published Books:   %v\n", obj["publishedBooks"]))
	b.WriteString(fmt.Sprintf("  Total Copies Sold: %v\n", obj["totalCopiesSold"]))
	b.WriteString(fmt.Sprintf("  Total Earned:      Rs.%v\n", obj["totalRoyaltyEarned"]))
	b.WriteString(fmt.Sprintf("  Royalty Paid:      Rs.%v\n", obj["royaltyPaid"]))
	b.WriteString(fmt.Sprintf("  Royalty Pending:   Rs.%v\n", obj["royaltyPending"]))
	b.WriteString(fmt.Sprintf("  Open Tickets:      %v\n", obj["openTickets"]))

	recent, _ := obj["recentBooks"].([]any)
	if len(recent) > 0 {
		b.WriteString(fmt.Sprintf("\n  \033[1mRecent Books:\033[0m\n"))
		for _, rb := range recent {
			book := rb.(map[string]any)
			b.WriteString(fmt.Sprintf("    \033[36m*\033[0m %s \033[90m(%s)\033[0m\n", str(book["title"]), str(book["status"])))
		}
	}

	b.WriteString("\n")
	return b.String()
}

func formatDefault(raw any) string {
	switch v := raw.(type) {
	case string:
		return v
	case map[string]any:
		var b strings.Builder
		b.WriteString("\n")
		for k, val := range v {
			b.WriteString(fmt.Sprintf("  %s  %v\n", k+":", val))
		}
		b.WriteString("\n")
		return b.String()
	default:
		pretty, _ := json.MarshalIndent(raw, "", "  ")
		return string(pretty)
	}
}

func str(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

func formatTime(s string) string {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return s
	}
	return t.Format("Jan 02, 2006 15:04")
}
