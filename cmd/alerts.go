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

func NewAlertsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "alerts",
		Short: "Manage alerts",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List alerts with filtering options",
		Long: `List alerts with various filtering options.
By default shows only active alerts (excluding silenced/inhibited).

Examples:
  alertctl alerts list                 # Show active alerts
  alertctl alerts list -A              # Show all alerts
  alertctl alerts list -s              # Show only silenced alerts
  alertctl alerts list -i              # Show only inhibited alerts
  alertctl alerts list -l critical     # Show alerts with severity=critical
  alertctl alerts list -A -l warning   # Show all alerts with severity=warning`,
		Run: listAlerts,
	}

	// Define flags with non-conflicting shorthands
	listCmd.Flags().BoolP("all", "A", false, "Show all alerts (including silenced/inhibited)")
	listCmd.Flags().BoolP("silenced", "s", false, "Show only silenced alerts")
	listCmd.Flags().BoolP("inhibited", "i", false, "Show only inhibited alerts")
	listCmd.Flags().Bool("active", false, "Show only active alerts (default)")
	listCmd.Flags().StringP("severity", "l", "", "Filter by severity level (e.g. 'critical')")
	listCmd.Flags().StringP("instance", "n", "", "Filter by instance name")

	cmd.AddCommand(listCmd)

	return cmd
}

func listAlerts(cmd *cobra.Command, args []string) {
	client := api.NewAlertManagerClient(alertmanagerURL, apiVersion)
	alerts, err := client.GetAlerts()
	if err != nil {
		fmt.Printf("Error getting alerts: %v\n", err)
		os.Exit(1)
	}

	// Get all filter flags
	filters := getFilters(cmd)
	filteredAlerts := applyFilters(alerts, filters)

	if len(filteredAlerts) == 0 {
		printNoAlertsMessage(filters)
		return
	}

	printAlerts(filteredAlerts, filters.showAll)
}

type alertFilters struct {
	showAll       bool
	showSilenced  bool
	showInhibited bool
	showActive    bool
	severity      string
	instance      string
}

func getFilters(cmd *cobra.Command) alertFilters {
	return alertFilters{
		showAll:       cmd.Flag("all").Value.String() == "true",
		showSilenced:  cmd.Flag("silenced").Value.String() == "true",
		showInhibited: cmd.Flag("inhibited").Value.String() == "true",
		showActive:    cmd.Flag("active").Value.String() == "true",
		severity:      cmd.Flag("severity").Value.String(),
		instance:      cmd.Flag("instance").Value.String(),
	}
}

func applyFilters(alerts []types.Alert, filters alertFilters) []types.Alert {
	var result []types.Alert

	for _, alert := range alerts {
		// Apply basic filters
		if filters.severity != "" && alert.Labels["severity"] != filters.severity {
			continue
		}
		if filters.instance != "" && alert.Labels["instance"] != filters.instance {
			continue
		}

		// Determine alert state
		state := getAlertState(alert)

		// Apply state filters
		switch {
		case filters.showAll:
			result = append(result, alert)
		case filters.showSilenced && state == "silenced":
			result = append(result, alert)
		case filters.showInhibited && state == "inhibited":
			result = append(result, alert)
		case filters.showActive && state == "active":
			result = append(result, alert)
		case !filters.showAll && !filters.showSilenced && !filters.showInhibited && !filters.showActive && state == "active":
			// Default case - show only active
			result = append(result, alert)
		}
	}

	return result
}

func getAlertState(alert types.Alert) string {
	if alert.AlertStatus == nil {
		return "active"
	}

	switch {
	case len(alert.AlertStatus.SilencedBy) > 0 || len(alert.AlertStatus.MutedBy) > 0:
		return "silenced"
	case len(alert.AlertStatus.InhibitedBy) > 0:
		return "inhibited"
	default:
		return "active"
	}
}

func printNoAlertsMessage(filters alertFilters) {
	var states []string
	if filters.showSilenced {
		states = append(states, "silenced")
	}
	if filters.showInhibited {
		states = append(states, "inhibited")
	}
	if filters.showActive || (!filters.showAll && len(states) == 0) {
		states = append(states, "active")
	}

	msg := "No alerts found"
	if len(states) > 0 {
		msg += " in state: " + strings.Join(states, " or ")
	}

	if filters.severity != "" {
		msg += fmt.Sprintf(" with severity '%s'", filters.severity)
	}
	if filters.instance != "" {
		msg += fmt.Sprintf(" from instance '%s'", filters.instance)
	}

	fmt.Println(msg)
}

func printAlerts(alerts []types.Alert, showDetails bool) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	defer w.Flush()

	headers := []string{"ALERT", "SEVERITY", "STATE", "SINCE", "INSTANCE", "SUMMARY"}
	if showDetails {
		headers = append(headers, "SILENCED BY", "INHIBITED BY")
	}
	fmt.Fprintln(w, strings.Join(headers, "\t"))

	for _, alert := range alerts {
		row := []string{
			getLabel(alert.Labels, "alertname"),
			getLabel(alert.Labels, "severity"),
			getAlertState(alert),
			formatDuration(time.Since(alert.StartsAt)),
			getLabel(alert.Labels, "instance"),
			getAnnotation(alert.Annotations, "summary", "description"),
		}

		if showDetails {
			silencedBy := strings.Join(alert.AlertStatus.SilencedBy, ",")
			inhibitedBy := strings.Join(alert.AlertStatus.InhibitedBy, ",")
			row = append(row, silencedBy, inhibitedBy)
		}

		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
}

// Helper functions
func getLabel(labels map[string]string, key string) string {
	if val, ok := labels[key]; ok {
		return val
	}
	return "-"
}

func getAnnotation(annotations map[string]string, keys ...string) string {
	for _, key := range keys {
		if val, ok := annotations[key]; ok {
			return val
		}
	}
	return "-"
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return d.Round(time.Second).String()
	}
	if d < time.Hour {
		return d.Round(time.Minute).String()
	}
	return d.Round(time.Hour).String()
}
