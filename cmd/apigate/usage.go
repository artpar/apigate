package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/artpar/apigate/adapters/sqlite"
	"github.com/spf13/cobra"
)

var usageCmd = &cobra.Command{
	Use:   "usage",
	Short: "View usage statistics",
	Long: `View API usage statistics for users.

Examples:
  apigate usage summary --user=user_123
  apigate usage summary --email=dev@example.com
  apigate usage history --user=user_123 --periods=6
  apigate usage recent --user=user_123 --limit=20`,
}

var usageSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Show usage summary for current period",
	RunE:  runUsageSummary,
}

var usageHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "Show usage history",
	RunE:  runUsageHistory,
}

var usageRecentCmd = &cobra.Command{
	Use:   "recent",
	Short: "Show recent requests",
	RunE:  runUsageRecent,
}

var (
	usageUserID  string
	usageEmail   string
	usagePeriods int
	usageLimit   int
)

func init() {
	rootCmd.AddCommand(usageCmd)

	usageCmd.AddCommand(usageSummaryCmd)
	usageCmd.AddCommand(usageHistoryCmd)
	usageCmd.AddCommand(usageRecentCmd)

	// Common flags
	usageSummaryCmd.Flags().StringVar(&usageUserID, "user", "", "user ID")
	usageSummaryCmd.Flags().StringVar(&usageEmail, "email", "", "user email")

	usageHistoryCmd.Flags().StringVar(&usageUserID, "user", "", "user ID")
	usageHistoryCmd.Flags().StringVar(&usageEmail, "email", "", "user email")
	usageHistoryCmd.Flags().IntVar(&usagePeriods, "periods", 6, "number of periods to show")

	usageRecentCmd.Flags().StringVar(&usageUserID, "user", "", "user ID")
	usageRecentCmd.Flags().StringVar(&usageEmail, "email", "", "user email")
	usageRecentCmd.Flags().IntVar(&usageLimit, "limit", 20, "number of requests to show")
}

func resolveUserID(db *sqlite.DB) (string, error) {
	if usageUserID != "" {
		return usageUserID, nil
	}
	if usageEmail != "" {
		userStore := sqlite.NewUserStore(db)
		user, err := userStore.GetByEmail(context.Background(), usageEmail)
		if err != nil {
			return "", fmt.Errorf("user not found: %s", usageEmail)
		}
		return user.ID, nil
	}
	return "", fmt.Errorf("either --user or --email is required")
}

func runUsageSummary(cmd *cobra.Command, args []string) error {
	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	userID, err := resolveUserID(db)
	if err != nil {
		return err
	}

	usageStore := sqlite.NewUsageStore(db)
	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	summary, err := usageStore.GetSummary(context.Background(), userID, start, now)
	if err != nil {
		return fmt.Errorf("failed to get usage: %w", err)
	}

	fmt.Printf("Usage Summary for %s\n", userID)
	fmt.Printf("Period: %s to %s\n\n", start.Format("2006-01-02"), now.Format("2006-01-02"))
	fmt.Printf("Requests:      %d\n", summary.RequestCount)
	fmt.Printf("Compute Units: %.2f\n", summary.ComputeUnits)
	fmt.Printf("Bytes In:      %d\n", summary.BytesIn)
	fmt.Printf("Bytes Out:     %d\n", summary.BytesOut)
	fmt.Printf("Errors:        %d\n", summary.ErrorCount)
	fmt.Printf("Avg Latency:   %d ms\n", summary.AvgLatencyMs)

	return nil
}

func runUsageHistory(cmd *cobra.Command, args []string) error {
	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	userID, err := resolveUserID(db)
	if err != nil {
		return err
	}

	usageStore := sqlite.NewUsageStore(db)
	summaries, err := usageStore.GetHistory(context.Background(), userID, usagePeriods)
	if err != nil {
		return fmt.Errorf("failed to get usage history: %w", err)
	}

	if len(summaries) == 0 {
		fmt.Println("No usage history found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PERIOD\tREQUESTS\tCOMPUTE\tBYTES IN\tBYTES OUT\tERRORS\tAVG LATENCY")
	fmt.Fprintln(w, "------\t--------\t-------\t--------\t---------\t------\t-----------")

	for _, s := range summaries {
		fmt.Fprintf(w, "%s\t%d\t%.2f\t%d\t%d\t%d\t%d ms\n",
			s.PeriodStart.Format("2006-01"),
			s.RequestCount,
			s.ComputeUnits,
			s.BytesIn,
			s.BytesOut,
			s.ErrorCount,
			s.AvgLatencyMs,
		)
	}

	w.Flush()
	return nil
}

func runUsageRecent(cmd *cobra.Command, args []string) error {
	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	userID, err := resolveUserID(db)
	if err != nil {
		return err
	}

	usageStore := sqlite.NewUsageStore(db)
	events, err := usageStore.GetRecentRequests(context.Background(), userID, usageLimit)
	if err != nil {
		return fmt.Errorf("failed to get recent requests: %w", err)
	}

	if len(events) == 0 {
		fmt.Println("No recent requests found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TIMESTAMP\tMETHOD\tPATH\tSTATUS\tLATENCY")
	fmt.Fprintln(w, "---------\t------\t----\t------\t-------")

	for _, e := range events {
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%d ms\n",
			e.Timestamp.Format("2006-01-02 15:04:05"),
			e.Method,
			e.Path,
			e.StatusCode,
			e.LatencyMs,
		)
	}

	w.Flush()
	return nil
}
