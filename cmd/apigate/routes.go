package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/artpar/apigate/adapters/sqlite"
	"github.com/artpar/apigate/domain/route"
	"github.com/spf13/cobra"
)

var routesCmd = &cobra.Command{
	Use:   "routes",
	Short: "Manage routes",
	Long: `Manage API routes for proxying requests.

Routes define how incoming requests are matched and forwarded to upstreams.

Examples:
  apigate routes list
  apigate routes get <route-id>
  apigate routes create --name="API v1" --path="/api/v1/*" --upstream=default
  apigate routes delete <route-id>
  apigate routes enable <route-id>
  apigate routes disable <route-id>`,
}

var routesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all routes",
	RunE:  runRoutesList,
}

var routesGetCmd = &cobra.Command{
	Use:   "get <route-id>",
	Short: "Get route details",
	Args:  cobra.ExactArgs(1),
	RunE:  runRoutesGet,
}

var routesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new route",
	RunE:  runRoutesCreate,
}

var routesDeleteCmd = &cobra.Command{
	Use:   "delete <route-id>",
	Short: "Delete a route",
	Args:  cobra.ExactArgs(1),
	RunE:  runRoutesDelete,
}

var routesEnableCmd = &cobra.Command{
	Use:   "enable <route-id>",
	Short: "Enable a route",
	Args:  cobra.ExactArgs(1),
	RunE:  runRoutesEnable,
}

var routesDisableCmd = &cobra.Command{
	Use:   "disable <route-id>",
	Short: "Disable a route",
	Args:  cobra.ExactArgs(1),
	RunE:  runRoutesDisable,
}

var (
	routeName        string
	routePath        string
	routeMatchType   string
	routeMethods     string
	routeUpstream    string
	routePathRewrite string
	routePriority    int
	routeProtocol    string
)

func init() {
	rootCmd.AddCommand(routesCmd)

	routesCmd.AddCommand(routesListCmd)
	routesCmd.AddCommand(routesGetCmd)
	routesCmd.AddCommand(routesCreateCmd)
	routesCmd.AddCommand(routesDeleteCmd)
	routesCmd.AddCommand(routesEnableCmd)
	routesCmd.AddCommand(routesDisableCmd)

	// Create command flags
	routesCreateCmd.Flags().StringVar(&routeName, "name", "", "route name (required)")
	routesCreateCmd.Flags().StringVar(&routePath, "path", "", "path pattern (required)")
	routesCreateCmd.Flags().StringVar(&routeMatchType, "match", "prefix", "match type: exact, prefix, regex")
	routesCreateCmd.Flags().StringVar(&routeMethods, "methods", "", "HTTP methods (comma-separated, empty = all)")
	routesCreateCmd.Flags().StringVar(&routeUpstream, "upstream", "", "upstream ID (required)")
	routesCreateCmd.Flags().StringVar(&routePathRewrite, "rewrite", "", "path rewrite expression")
	routesCreateCmd.Flags().IntVar(&routePriority, "priority", 0, "route priority (higher = first)")
	routesCreateCmd.Flags().StringVar(&routeProtocol, "protocol", "http", "protocol: http, http_stream, sse, websocket")
	routesCreateCmd.MarkFlagRequired("name")
	routesCreateCmd.MarkFlagRequired("path")
	routesCreateCmd.MarkFlagRequired("upstream")
}

func runRoutesList(cmd *cobra.Command, args []string) error {
	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	routeStore := sqlite.NewRouteStore(db)
	routes, err := routeStore.List(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list routes: %w", err)
	}

	if len(routes) == 0 {
		fmt.Println("No routes found.")
		fmt.Println()
		fmt.Println("Create a route with: apigate routes create --name=\"My Route\" --path=\"/api/*\" --upstream=default")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tPATH\tMATCH\tMETHODS\tUPSTREAM\tPRIORITY\tENABLED")
	fmt.Fprintln(w, "--\t----\t----\t-----\t-------\t--------\t--------\t-------")

	for _, r := range routes {
		methods := "*"
		if len(r.Methods) > 0 {
			methods = strings.Join(r.Methods, ",")
		}
		enabled := "no"
		if r.Enabled {
			enabled = "yes"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%d\t%s\n",
			r.ID, r.Name, r.PathPattern, r.MatchType, methods, r.UpstreamID, r.Priority, enabled)
	}

	w.Flush()
	return nil
}

func runRoutesGet(cmd *cobra.Command, args []string) error {
	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	routeStore := sqlite.NewRouteStore(db)
	r, err := routeStore.Get(context.Background(), args[0])
	if err != nil {
		return fmt.Errorf("route not found: %s", args[0])
	}

	fmt.Printf("ID:          %s\n", r.ID)
	fmt.Printf("Name:        %s\n", r.Name)
	if r.Description != "" {
		fmt.Printf("Description: %s\n", r.Description)
	}
	fmt.Printf("Path:        %s\n", r.PathPattern)
	fmt.Printf("Match Type:  %s\n", r.MatchType)
	methods := "*"
	if len(r.Methods) > 0 {
		methods = strings.Join(r.Methods, ", ")
	}
	fmt.Printf("Methods:     %s\n", methods)
	fmt.Printf("Upstream:    %s\n", r.UpstreamID)
	if r.PathRewrite != "" {
		fmt.Printf("Path Rewrite: %s\n", r.PathRewrite)
	}
	if r.MethodOverride != "" {
		fmt.Printf("Method Override: %s\n", r.MethodOverride)
	}
	fmt.Printf("Protocol:    %s\n", r.Protocol)
	fmt.Printf("Priority:    %d\n", r.Priority)
	fmt.Printf("Enabled:     %v\n", r.Enabled)
	fmt.Printf("Created:     %s\n", r.CreatedAt.Format("2006-01-02 15:04:05"))
	if !r.UpdatedAt.IsZero() {
		fmt.Printf("Updated:     %s\n", r.UpdatedAt.Format("2006-01-02 15:04:05"))
	}

	// Show transforms if present
	if r.RequestTransform != nil {
		data, _ := json.MarshalIndent(r.RequestTransform, "", "  ")
		fmt.Printf("Request Transform:\n%s\n", string(data))
	}
	if r.ResponseTransform != nil {
		data, _ := json.MarshalIndent(r.ResponseTransform, "", "  ")
		fmt.Printf("Response Transform:\n%s\n", string(data))
	}
	if r.MeteringExpr != "" {
		fmt.Printf("Metering:    %s (%s)\n", r.MeteringExpr, r.MeteringMode)
	}

	return nil
}

func runRoutesCreate(cmd *cobra.Command, args []string) error {
	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	// Parse methods
	var methods []string
	if routeMethods != "" {
		methods = strings.Split(routeMethods, ",")
		for i := range methods {
			methods[i] = strings.TrimSpace(strings.ToUpper(methods[i]))
		}
	}

	// Parse match type
	matchType := route.MatchPrefix
	switch strings.ToLower(routeMatchType) {
	case "exact":
		matchType = route.MatchExact
	case "regex":
		matchType = route.MatchRegex
	case "prefix":
		matchType = route.MatchPrefix
	default:
		return fmt.Errorf("invalid match type: %s (use: exact, prefix, regex)", routeMatchType)
	}

	// Parse protocol
	protocol := route.ProtocolHTTP
	switch strings.ToLower(routeProtocol) {
	case "http":
		protocol = route.ProtocolHTTP
	case "http_stream":
		protocol = route.ProtocolHTTPStream
	case "sse":
		protocol = route.ProtocolSSE
	case "websocket":
		protocol = route.ProtocolWebSocket
	default:
		return fmt.Errorf("invalid protocol: %s", routeProtocol)
	}

	r := route.Route{
		ID:          fmt.Sprintf("route_%d", os.Getpid()),
		Name:        routeName,
		PathPattern: routePath,
		MatchType:   matchType,
		Methods:     methods,
		UpstreamID:  routeUpstream,
		PathRewrite: routePathRewrite,
		Protocol:    protocol,
		Priority:    routePriority,
		Enabled:     true,
	}

	routeStore := sqlite.NewRouteStore(db)
	if err := routeStore.Create(context.Background(), r); err != nil {
		return fmt.Errorf("failed to create route: %w", err)
	}

	fmt.Printf("%s Created route: %s\n", checkMark, r.ID)
	fmt.Printf("   Name:     %s\n", r.Name)
	fmt.Printf("   Path:     %s\n", r.PathPattern)
	fmt.Printf("   Upstream: %s\n", r.UpstreamID)

	return nil
}

func runRoutesDelete(cmd *cobra.Command, args []string) error {
	routeID := args[0]

	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	routeStore := sqlite.NewRouteStore(db)

	// Check if route exists
	r, err := routeStore.Get(context.Background(), routeID)
	if err != nil {
		return fmt.Errorf("route not found: %s", routeID)
	}

	// Confirm deletion
	if !confirm(fmt.Sprintf("Delete route %s (%s)?", r.Name, routeID)) {
		fmt.Println("Aborted.")
		return nil
	}

	if err := routeStore.Delete(context.Background(), routeID); err != nil {
		return fmt.Errorf("failed to delete route: %w", err)
	}

	fmt.Printf("%s Deleted route: %s\n", checkMark, routeID)
	return nil
}

func runRoutesEnable(cmd *cobra.Command, args []string) error {
	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	routeStore := sqlite.NewRouteStore(db)
	r, err := routeStore.Get(context.Background(), args[0])
	if err != nil {
		return fmt.Errorf("route not found: %s", args[0])
	}

	if r.Enabled {
		fmt.Printf("Route %s is already enabled\n", r.Name)
		return nil
	}

	r.Enabled = true
	if err := routeStore.Update(context.Background(), r); err != nil {
		return fmt.Errorf("failed to enable route: %w", err)
	}

	fmt.Printf("%s Enabled route: %s (%s)\n", checkMark, r.Name, r.ID)
	return nil
}

func runRoutesDisable(cmd *cobra.Command, args []string) error {
	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	routeStore := sqlite.NewRouteStore(db)
	r, err := routeStore.Get(context.Background(), args[0])
	if err != nil {
		return fmt.Errorf("route not found: %s", args[0])
	}

	if !r.Enabled {
		fmt.Printf("Route %s is already disabled\n", r.Name)
		return nil
	}

	r.Enabled = false
	if err := routeStore.Update(context.Background(), r); err != nil {
		return fmt.Errorf("failed to disable route: %w", err)
	}

	fmt.Printf("%s Disabled route: %s (%s)\n", checkMark, r.Name, r.ID)
	return nil
}
