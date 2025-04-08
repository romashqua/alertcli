package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"alertctl/internal/api"
	"alertctl/internal/types"

	"github.com/spf13/cobra"
)

func NewSilencesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "silences",
		Short: "Manage Alertmanager silences",
	}

	cmd.AddCommand(
		newSilencesListCmd(),
		newSilencesCreateCmd(),
		newSilencesDeleteCmd(),
	)

	return cmd
}

func newSilencesListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all silences",
		Run:   listSilences,
	}
}

func newSilencesCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [flags]",
		Short: "Create new silence",
		Run:   createSilence,
	}

	cmd.Flags().String("creator", "alertctl", "Silence creator")
	cmd.Flags().String("comment", "", "Silence comment (required)")
	cmd.Flags().Duration("duration", 2*time.Hour, "Silence duration")
	cmd.Flags().StringSlice("matcher", []string{}, "Matchers in format 'name=value' or 'name=~regex'")
	cmd.Flags().String("alertname", "", "Alert name to silence (shortcut)")
	cmd.Flags().String("instance", "", "Instance to silence (shortcut)")

	cmd.MarkFlagRequired("comment")

	return cmd
}

func newSilencesDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete <id>",
		Short:   "Delete silence by ID",
		Args:    cobra.ExactArgs(1),
		Example: "alertctl silences delete abc123",
		Run:     deleteSilence,
	}
}

func listSilences(cmd *cobra.Command, args []string) {
	client := api.NewAlertManagerClient(alertmanagerURL, apiVersion)
	silences, err := client.GetSilences()
	if err != nil {
		fmt.Printf("Error getting silences: %v\n", err)
		os.Exit(1)
	}

	if len(silences) == 0 {
		fmt.Println("No active silences found")
		return
	}

	printSilences(silences)
}

func createSilence(cmd *cobra.Command, args []string) {
	creator, _ := cmd.Flags().GetString("creator")
	comment, _ := cmd.Flags().GetString("comment")
	duration, _ := cmd.Flags().GetDuration("duration")
	matcherFlags, _ := cmd.Flags().GetStringSlice("matcher")
	alertName, _ := cmd.Flags().GetString("alertname")
	instance, _ := cmd.Flags().GetString("instance")

	// Build matchers from flags
	matchers := []types.Matcher{}

	// Add shortcut matchers if provided
	if alertName != "" {
		matchers = append(matchers, types.Matcher{
			Name:    "alertname",
			Value:   alertName,
			IsRegex: false,
		})
	}

	if instance != "" {
		matchers = append(matchers, types.Matcher{
			Name:    "instance",
			Value:   instance,
			IsRegex: false,
		})
	}

	// Parse matcher flags
	for _, m := range matcherFlags {
		matcher, err := parseMatcher(m)
		if err != nil {
			fmt.Printf("Invalid matcher format '%s': %v\n", m, err)
			os.Exit(1)
		}
		matchers = append(matchers, matcher)
	}

	// Validate at least one matcher exists
	if len(matchers) == 0 {
		fmt.Println("Error: at least one matcher must be specified")
		fmt.Println("Use --matcher or --alertname/--instance flags")
		os.Exit(1)
	}

	silence := types.Silence{
		Matchers:  matchers,
		StartsAt:  time.Now(),
		EndsAt:    time.Now().Add(duration),
		CreatedBy: creator,
		Comment:   comment,
	}

	client := api.NewAlertManagerClient(alertmanagerURL, apiVersion)
	id, err := client.CreateSilence(silence)
	if err != nil {
		fmt.Printf("Error creating silence: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully created silence with ID: %s\n", id)
}

func deleteSilence(cmd *cobra.Command, args []string) {
	client := api.NewAlertManagerClient(alertmanagerURL, apiVersion)
	err := client.DeleteSilence(args[0])
	if err != nil {
		fmt.Printf("Error deleting silence: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully deleted silence %s\n", args[0])
}

func printSilences(silences []types.Silence) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSTATUS\tSTART\tEND\tCREATOR\tCOMMENT\tMATCHERS")

	for _, silence := range silences {
		var matcherStrings []string
		for _, m := range silence.Matchers {
			op := "="
			if m.IsRegex {
				op = "=~"
			}
			matcherStrings = append(matcherStrings, fmt.Sprintf("%s%s%s", m.Name, op, m.Value))
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			silence.ID,
			silence.GetState(),
			silence.StartsAt.Format(time.RFC822),
			silence.EndsAt.Format(time.RFC822),
			silence.CreatedBy,
			shortenString(silence.Comment, 30),
			strings.Join(matcherStrings, ","),
		)
	}
	w.Flush()
}

// Helper functions

func parseMatcher(input string) (types.Matcher, error) {
	parts := strings.SplitN(input, "=", 2)
	if len(parts) != 2 {
		return types.Matcher{}, fmt.Errorf("invalid matcher format, expected name=value")
	}

	name := parts[0]
	value := parts[1]
	isRegex := false

	// Handle regex matchers (name=~regex)
	if strings.HasPrefix(value, "~") {
		isRegex = true
		value = strings.TrimPrefix(value, "~")
	}

	return types.Matcher{
		Name:    strings.TrimSpace(name),
		Value:   strings.TrimSpace(value),
		IsRegex: isRegex,
	}, nil
}

func shortenString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
