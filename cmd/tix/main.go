package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/joe-williamson/tix/internal/config"
	"github.com/joe-williamson/tix/internal/srebr"
)

func main() {
	root := &cobra.Command{
		Use:   "tix",
		Short: "SRE ticket tools",
		Example: `  tix bg c9-prod ESS-46119
  tix bg prod-cluster ESS-46121 --hours 24
  tix list
  tix info ESS-47181`,
	}

	root.AddCommand(bgCmd(), listCmd(), infoCmd())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

// ── bg ───────────────────────────────────────────────────────────────────────

var (
	flagHours    int
	flagUser     string
	flagGroup    string
	flagProject  string
	flagNS       string
	flagProvider string
	flagDryRun   bool
)

func bgCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bg <profile> <source-ticket> [source-ticket...]",
		Short: "Create a SREBR breakglass ticket",
		Example: `  tix bg c9-prod ESS-46119
  tix bg prod-cluster ESS-46121 --hours 24
  tix bg tlm-prod ESS-46120 --dry-run`,
		Args: cobra.MinimumNArgs(2),
		RunE: runBG,
	}
	cmd.Flags().IntVar(&flagHours, "hours", 0, "Override hours (4, 8, 12, 24, 48)")
	cmd.Flags().StringVar(&flagUser, "user", "", "Override breakglass user")
	cmd.Flags().StringVar(&flagGroup, "group", "", "Override breakglass group")
	cmd.Flags().StringVar(&flagProject, "project", "", "Override GCP project")
	cmd.Flags().StringVar(&flagNS, "namespace", "", "Override GKE namespace")
	cmd.Flags().StringVar(&flagProvider, "provider", "", "Override provider (foxpass, entra, gcp)")
	cmd.Flags().BoolVar(&flagDryRun, "dry-run", false, "Show what would be created without creating")
	return cmd
}

func runBG(cmd *cobra.Command, args []string) error {
	profileName := args[0]
	sources := args[1:]

	defaults, profiles, err := config.Load()
	if err != nil {
		return err
	}

	overrides := config.Profile{
		User:     flagUser,
		Hours:    flagHours,
		Group:    flagGroup,
		Project:  flagProject,
		Namespace: flagNS,
		Provider: flagProvider,
	}
	p, err := config.Resolve(profileName, defaults, profiles, overrides)
	if err != nil {
		return err
	}

	if flagDryRun {
		fmt.Println("DRY RUN — would create:")
		fmt.Printf("  Summary:     %s\n", srebr.BuildSummary(p))
		fmt.Println("  Description:")
		for _, line := range strings.Split(strings.TrimRight(srebr.BuildDescription(p), "\n"), "\n") {
			fmt.Printf("               %s\n", line)
		}
		fmt.Printf("  Hours:       %d\n", p.Hours)
		fmt.Printf("  User:        %s\n", p.User)
		fmt.Printf("  Link to:     %s\n", strings.Join(sources, ", "))
		return nil
	}

	creds, err := config.LoadJiraCreds()
	if err != nil {
		return err
	}

	_, err = srebr.CreateTicket(context.Background(), creds, p, sources)
	return err
}

// ── list ─────────────────────────────────────────────────────────────────────

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available breakglass profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			defaults, profiles, err := config.Load()
			if err != nil {
				return err
			}

			fmt.Printf("Available profiles (default %d hours):\n\n", defaults.Hours)

			names := make([]string, 0, len(profiles))
			for n := range profiles {
				names = append(names, n)
			}
			sort.Strings(names)

			for _, n := range names {
				p := profiles[n]
				if p.User == "" {
					p.User = defaults.User
				}
				if p.Env == "" {
					p.Env = defaults.Env
				}
				if p.Hours == 0 {
					p.Hours = defaults.Hours
				}
				fmt.Printf("  %-18s %s\n", n, srebr.BuildSummary(p))
			}
			fmt.Println()
			return nil
		},
	}
}

// ── info ─────────────────────────────────────────────────────────────────────

func infoCmd() *cobra.Command {
	var showComments bool
	cmd := &cobra.Command{
		Use:   "info <ticket-key> [ticket-key...]",
		Short: "Inspect one or more existing Jira tickets",
		Example: `  tix info ESS-47181
  tix info ESS-47181 --comments
  tix info SREBR-20015 ESS-46988`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			creds, err := config.LoadJiraCreds()
			if err != nil {
				return err
			}
			return srebr.InspectTickets(context.Background(), creds, args, showComments)
		},
	}
	cmd.Flags().BoolVar(&showComments, "comments", false, "Also display ticket comments")
	return cmd
}
