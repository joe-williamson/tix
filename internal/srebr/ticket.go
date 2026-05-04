// Package srebr implements SREBR breakglass ticket creation and inspection.
package srebr

import (
	"context"
	"fmt"
	"strings"

	"github.com/symphony/srebr-bg/internal/config"
)

const (
	project = "SREBR"
	jiraURL = "https://perzoinc.atlassian.net"
)

// hoursMap maps allowed hour values to their Jira customfield option IDs.
var hoursMap = map[int]string{
	4:  "16404",
	8:  "16405",
	12: "16406",
	24: "41469",
	48: "41468",
}

// envMap maps environment names to their Jira customfield option IDs.
var envMap = map[string]string{
	"prod": "16416",
}

// CreateResult holds the outcome of a successful ticket creation.
type CreateResult struct {
	Key string
	URL string
}

// BuildSummary constructs the SREBR ticket summary from a profile.
// Matches the formats validated in the existing bg_ticket.py tool.
func BuildSummary(p config.Profile) string {
	switch p.Provider {
	case "gcp":
		return fmt.Sprintf("SRE Breakglass Access - GKE - %s - %s - %d hours",
			p.Project, p.User, p.Hours)
	case "entra":
		return fmt.Sprintf("Breakglass Access - %s - %s", strings.ToUpper(p.Env), p.Group)
	default: // foxpass
		prefix := p.SummaryPrefix
		if prefix == "" {
			prefix = "SRE Breakglass Access"
		}
		return fmt.Sprintf("%s - %s - %s - %d hours", prefix, strings.ToUpper(p.Env), p.User, p.Hours)
	}
}

// BuildDescription constructs the SREBR ticket description from a profile.
func BuildDescription(p config.Profile) string {
	lines := []string{
		"provider: " + p.Provider,
		"group: " + p.Group,
	}
	if p.Provider == "gcp" {
		lines = append(lines, "project: "+p.Project)
		ns := p.Namespace
		if ns == "" {
			ns = "unused"
		}
		lines = append(lines, "namespace: "+ns)
	}
	return strings.Join(lines, "\n") + "\n"
}

// CreateTicket creates a SREBR ticket, links it to the source tickets, and approves it.
func CreateTicket(ctx context.Context, creds config.Creds, p config.Profile, sources []string) (*CreateResult, error) {
	hoursID, ok := hoursMap[p.Hours]
	if !ok {
		valid := make([]string, 0, len(hoursMap))
		for h := range hoursMap {
			valid = append(valid, fmt.Sprintf("%d", h))
		}
		return nil, fmt.Errorf("invalid hours %d; valid values: %s", p.Hours, strings.Join(valid, ", "))
	}

	env := strings.ToLower(p.Env)
	envID, ok := envMap[env]
	if !ok {
		envID = envMap["prod"]
	}

	fields := map[string]any{
		"project":          map[string]any{"key": project},
		"summary":          BuildSummary(p),
		"description":      BuildDescription(p),
		"issuetype":        map[string]any{"name": "Task"},
		"customfield_15390": p.User,
		"customfield_15555": map[string]any{"id": envID},
		"customfield_15542": map[string]any{"id": hoursID},
	}

	client := NewClient(creds)

	key, err := client.CreateIssue(fields)
	if err != nil {
		return nil, fmt.Errorf("create ticket: %w", err)
	}

	url := fmt.Sprintf("%s/browse/%s", jiraURL, key)
	fmt.Printf("Created: %s\n", url)

	for _, src := range sources {
		if err := client.LinkIssues("Relates", src, key); err != nil {
			fmt.Printf("WARNING: could not link %s → %s: %v\n", src, key, err)
		}
	}

	if err := client.Transition(key, "Approve"); err != nil {
		fmt.Printf("WARNING: could not auto-approve %s (do it manually): %v\n", key, err)
	}

	return &CreateResult{Key: key, URL: url}, nil
}

// InspectTickets prints a formatted summary of each ticket key.
// If showComments is true, comments are also fetched and printed.
func InspectTickets(ctx context.Context, creds config.Creds, keys []string, showComments bool) error {
	client := NewClient(creds)
	sep := strings.Repeat("=", 60)

	for _, key := range keys {
		issue, err := client.GetIssue(key)
		if err != nil {
			fmt.Printf("%s\nERROR fetching %s: %v\n", sep, key, err)
			continue
		}

		user := "None"
		if issue.Fields.Assignee != nil {
			user = issue.Fields.Assignee.DisplayName
		}

		fmt.Println(sep)
		fmt.Printf("key:         %s\n", issue.Key)
		fmt.Printf("url:         %s/browse/%s\n", jiraURL, issue.Key)
		fmt.Printf("summary:     %s\n", issue.Fields.Summary)
		fmt.Printf("issuetype:   %s\n", issue.Fields.IssueType.Name)
		fmt.Printf("status:      %s\n", issue.Fields.Status.Name)
		fmt.Println("description:")
		for _, line := range strings.Split(strings.TrimRight(issue.Fields.Description, "\n"), "\n") {
			fmt.Printf("  %s\n", line)
		}
		fmt.Printf("user:        %s\n", user)

		if showComments {
			comments, err := client.GetComments(key)
			if err != nil {
				fmt.Printf("comments:    ERROR: %v\n", err)
				continue
			}
			if len(comments) == 0 {
				fmt.Println("comments:    (none)")
				continue
			}
			fmt.Printf("comments:    (%d)\n", len(comments))
			csep := strings.Repeat("-", 40)
			for _, c := range comments {
				fmt.Println(csep)
				fmt.Printf("  author:  %s\n", c.Author.DisplayName)
				fmt.Printf("  date:    %s\n", c.Created[:10])
				for _, line := range strings.Split(strings.TrimRight(c.Body, "\n"), "\n") {
					fmt.Printf("  %s\n", line)
				}
			}
		}
	}
	return nil
}
