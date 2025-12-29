package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/artpar/apigate/adapters/sqlite"
	"github.com/artpar/apigate/ports"
	"github.com/spf13/cobra"
)

var plansCmd = &cobra.Command{
	Use:   "plans",
	Short: "Manage subscription plans",
	Long: `Manage API subscription plans.

Plans define rate limits, quotas, and pricing for API access.

Examples:
  apigate plans list
  apigate plans get <plan-id>
  apigate plans create --id=pro --name="Pro" --rate-limit=600 --requests=100000
  apigate plans delete <plan-id>`,
}

var plansListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all plans",
	RunE:  runPlansList,
}

var plansGetCmd = &cobra.Command{
	Use:   "get <plan-id>",
	Short: "Get plan details",
	Args:  cobra.ExactArgs(1),
	RunE:  runPlansGet,
}

var plansCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new plan",
	RunE:  runPlansCreate,
}

var plansDeleteCmd = &cobra.Command{
	Use:   "delete <plan-id>",
	Short: "Delete a plan",
	Args:  cobra.ExactArgs(1),
	RunE:  runPlansDelete,
}

var plansEnableCmd = &cobra.Command{
	Use:   "enable <plan-id>",
	Short: "Enable a plan",
	Args:  cobra.ExactArgs(1),
	RunE:  runPlansEnable,
}

var plansDisableCmd = &cobra.Command{
	Use:   "disable <plan-id>",
	Short: "Disable a plan",
	Args:  cobra.ExactArgs(1),
	RunE:  runPlansDisable,
}

var (
	planID          string
	planName        string
	planDescription string
	planRateLimit   int
	planRequests    int64
	planPrice       int64
	planOverage     int64
	planDefault     bool
)

func init() {
	rootCmd.AddCommand(plansCmd)

	plansCmd.AddCommand(plansListCmd)
	plansCmd.AddCommand(plansGetCmd)
	plansCmd.AddCommand(plansCreateCmd)
	plansCmd.AddCommand(plansDeleteCmd)
	plansCmd.AddCommand(plansEnableCmd)
	plansCmd.AddCommand(plansDisableCmd)

	// Create command flags
	plansCreateCmd.Flags().StringVar(&planID, "id", "", "plan ID (required)")
	plansCreateCmd.Flags().StringVar(&planName, "name", "", "plan name (required)")
	plansCreateCmd.Flags().StringVar(&planDescription, "description", "", "plan description")
	plansCreateCmd.Flags().IntVar(&planRateLimit, "rate-limit", 60, "requests per minute")
	plansCreateCmd.Flags().Int64Var(&planRequests, "requests", 1000, "requests per month (-1 = unlimited)")
	plansCreateCmd.Flags().Int64Var(&planPrice, "price", 0, "monthly price in cents")
	plansCreateCmd.Flags().Int64Var(&planOverage, "overage", 0, "overage price in cents per request")
	plansCreateCmd.Flags().BoolVar(&planDefault, "default", false, "set as default plan")
	plansCreateCmd.MarkFlagRequired("id")
	plansCreateCmd.MarkFlagRequired("name")
}

func runPlansList(cmd *cobra.Command, args []string) error {
	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	planStore := sqlite.NewPlanStore(db)
	plans, err := planStore.List(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list plans: %w", err)
	}

	if len(plans) == 0 {
		fmt.Println("No plans found.")
		fmt.Println()
		fmt.Println("Create a plan with: apigate plans create --id=free --name=\"Free\" --rate-limit=60 --requests=1000")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tRATE LIMIT\tREQUESTS/MO\tPRICE\tDEFAULT\tENABLED")
	fmt.Fprintln(w, "--\t----\t----------\t-----------\t-----\t-------\t-------")

	for _, p := range plans {
		requests := fmt.Sprintf("%d", p.RequestsPerMonth)
		if p.RequestsPerMonth < 0 {
			requests = "unlimited"
		}
		price := fmt.Sprintf("$%.2f", float64(p.PriceMonthly)/100)
		if p.PriceMonthly == 0 {
			price = "free"
		}
		isDefault := ""
		if p.IsDefault {
			isDefault = "yes"
		}
		enabled := "no"
		if p.Enabled {
			enabled = "yes"
		}
		fmt.Fprintf(w, "%s\t%s\t%d/min\t%s\t%s\t%s\t%s\n",
			p.ID, p.Name, p.RateLimitPerMinute, requests, price, isDefault, enabled)
	}

	w.Flush()
	return nil
}

func runPlansGet(cmd *cobra.Command, args []string) error {
	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	planStore := sqlite.NewPlanStore(db)
	p, err := planStore.Get(context.Background(), args[0])
	if err != nil {
		return fmt.Errorf("plan not found: %s", args[0])
	}

	fmt.Printf("ID:              %s\n", p.ID)
	fmt.Printf("Name:            %s\n", p.Name)
	if p.Description != "" {
		fmt.Printf("Description:     %s\n", p.Description)
	}
	fmt.Printf("Rate Limit:      %d/min\n", p.RateLimitPerMinute)
	if p.RequestsPerMonth < 0 {
		fmt.Printf("Requests/Month:  unlimited\n")
	} else {
		fmt.Printf("Requests/Month:  %d\n", p.RequestsPerMonth)
	}
	fmt.Printf("Monthly Price:   $%.2f\n", float64(p.PriceMonthly)/100)
	if p.OveragePrice > 0 {
		fmt.Printf("Overage Price:   $%.4f/request\n", float64(p.OveragePrice)/100)
	}
	fmt.Printf("Default:         %v\n", p.IsDefault)
	fmt.Printf("Enabled:         %v\n", p.Enabled)
	if p.StripePriceID != "" {
		fmt.Printf("Stripe Price:    %s\n", p.StripePriceID)
	}
	if p.PaddlePriceID != "" {
		fmt.Printf("Paddle Price:    %s\n", p.PaddlePriceID)
	}
	if p.LemonVariantID != "" {
		fmt.Printf("Lemon Variant:   %s\n", p.LemonVariantID)
	}
	fmt.Printf("Created:         %s\n", p.CreatedAt.Format("2006-01-02 15:04:05"))

	return nil
}

func runPlansCreate(cmd *cobra.Command, args []string) error {
	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	p := ports.Plan{
		ID:                 planID,
		Name:               planName,
		Description:        planDescription,
		RateLimitPerMinute: planRateLimit,
		RequestsPerMonth:   planRequests,
		PriceMonthly:       planPrice,
		OveragePrice:       planOverage,
		IsDefault:          planDefault,
		Enabled:            true,
	}

	planStore := sqlite.NewPlanStore(db)
	if err := planStore.Create(context.Background(), p); err != nil {
		return fmt.Errorf("failed to create plan: %w", err)
	}

	fmt.Printf("%s Created plan: %s\n", checkMark, p.ID)
	fmt.Printf("   Name:         %s\n", p.Name)
	fmt.Printf("   Rate Limit:   %d/min\n", p.RateLimitPerMinute)
	fmt.Printf("   Requests/Mo:  %d\n", p.RequestsPerMonth)

	return nil
}

func runPlansDelete(cmd *cobra.Command, args []string) error {
	planID := args[0]

	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	planStore := sqlite.NewPlanStore(db)

	// Check if plan exists
	p, err := planStore.Get(context.Background(), planID)
	if err != nil {
		return fmt.Errorf("plan not found: %s", planID)
	}

	// Confirm deletion
	if !confirm(fmt.Sprintf("Delete plan %s (%s)?", p.Name, planID)) {
		fmt.Println("Aborted.")
		return nil
	}

	if err := planStore.Delete(context.Background(), planID); err != nil {
		return fmt.Errorf("failed to delete plan: %w", err)
	}

	fmt.Printf("%s Deleted plan: %s\n", checkMark, planID)
	return nil
}

func runPlansEnable(cmd *cobra.Command, args []string) error {
	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	planStore := sqlite.NewPlanStore(db)
	p, err := planStore.Get(context.Background(), args[0])
	if err != nil {
		return fmt.Errorf("plan not found: %s", args[0])
	}

	if p.Enabled {
		fmt.Printf("Plan %s is already enabled\n", p.Name)
		return nil
	}

	p.Enabled = true
	if err := planStore.Update(context.Background(), p); err != nil {
		return fmt.Errorf("failed to enable plan: %w", err)
	}

	fmt.Printf("%s Enabled plan: %s (%s)\n", checkMark, p.Name, p.ID)
	return nil
}

func runPlansDisable(cmd *cobra.Command, args []string) error {
	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	planStore := sqlite.NewPlanStore(db)
	p, err := planStore.Get(context.Background(), args[0])
	if err != nil {
		return fmt.Errorf("plan not found: %s", args[0])
	}

	if !p.Enabled {
		fmt.Printf("Plan %s is already disabled\n", p.Name)
		return nil
	}

	p.Enabled = false
	if err := planStore.Update(context.Background(), p); err != nil {
		return fmt.Errorf("failed to disable plan: %w", err)
	}

	fmt.Printf("%s Disabled plan: %s (%s)\n", checkMark, p.Name, p.ID)
	return nil
}
