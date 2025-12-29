// Package example demonstrates how to use the declarative module system.
// This is an example of how the core packages work together.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/artpar/apigate/core/convention"
	cliChannel "github.com/artpar/apigate/core/channel/cli"
	httpChannel "github.com/artpar/apigate/core/channel/http"
	"github.com/artpar/apigate/core/runtime"
	"github.com/artpar/apigate/core/schema"
	"github.com/artpar/apigate/core/storage"
	"github.com/spf13/cobra"
)

func main() {
	// Create storage
	store, err := storage.NewSQLiteStore("./example.db")
	if err != nil {
		log.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	// Create a storage adapter for runtime
	adapter := &storageAdapter{store: store}

	// Create runtime
	rt := runtime.New(adapter, runtime.Config{
		ModulesDir: "./modules",
	})

	// Create root CLI command
	rootCmd := &cobra.Command{
		Use:   "example",
		Short: "Example declarative API",
	}

	// Create and register CLI channel
	cli := cliChannel.New(rootCmd, rt)
	rt.RegisterChannel(cli)

	// Create and register HTTP channel
	http := httpChannel.New(rt, ":8080")
	rt.RegisterChannel(http)

	// Define modules programmatically or load from YAML
	modules := []schema.Module{
		{
			Name: "product",
			Schema: map[string]schema.Field{
				"name":        {Type: schema.FieldTypeString},
				"description": {Type: schema.FieldTypeString},
				"price":       {Type: schema.FieldTypeInt, Default: 0},
				"sku":         {Type: schema.FieldTypeString, Unique: true, Lookup: true},
				"category":    {Type: schema.FieldTypeRef, To: "category"},
				"enabled":     {Type: schema.FieldTypeBool, Default: true},
			},
			Actions: map[string]schema.Action{
				"enable":  {Set: map[string]string{"enabled": "true"}},
				"disable": {Set: map[string]string{"enabled": "false"}},
			},
			Channels: schema.Channels{
				HTTP: schema.HTTPChannel{
					Serve: schema.HTTPServe{Enabled: true},
				},
				CLI: schema.CLIChannel{
					Serve: schema.CLIServe{Enabled: true},
				},
			},
		},
		{
			Name: "category",
			Schema: map[string]schema.Field{
				"name":        {Type: schema.FieldTypeString},
				"description": {Type: schema.FieldTypeString},
				"slug":        {Type: schema.FieldTypeString, Unique: true, Lookup: true},
			},
			Channels: schema.Channels{
				HTTP: schema.HTTPChannel{
					Serve: schema.HTTPServe{Enabled: true},
				},
				CLI: schema.CLIChannel{
					Serve: schema.CLIServe{Enabled: true},
				},
			},
		},
	}

	// Load modules
	for _, mod := range modules {
		if err := rt.LoadModule(mod); err != nil {
			log.Fatalf("Failed to load module %s: %v", mod.Name, err)
		}
		fmt.Printf("Loaded module: %s\n", mod.Name)
	}

	// Print registered paths
	fmt.Println("\nRegistered HTTP paths:")
	for _, path := range rt.Registry().GetHTTPPaths() {
		fmt.Printf("  %s %s -> %s.%s\n", path.Method, path.Path, path.Module, path.Action)
	}

	fmt.Println("\nRegistered CLI paths:")
	for _, path := range rt.Registry().GetCLIPaths() {
		fmt.Printf("  %s -> %s.%s\n", path.Path, path.Module, path.Action)
	}

	// Start HTTP server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := rt.Start(ctx); err != nil {
		log.Fatalf("Failed to start runtime: %v", err)
	}

	fmt.Println("\nHTTP server starting on :8080")
	fmt.Println("Try: curl http://localhost:8080/api/products")

	// Wait for interrupt
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\nShutting down...")
	rt.Stop(ctx)
}

// storageAdapter adapts storage.Store to runtime.Storage
type storageAdapter struct {
	store *storage.SQLiteStore
}

func (a *storageAdapter) CreateTable(ctx context.Context, mod convention.Derived) error {
	return a.store.CreateTable(ctx, mod)
}

func (a *storageAdapter) Create(ctx context.Context, module string, data map[string]any) (string, error) {
	return a.store.Create(ctx, module, data)
}

func (a *storageAdapter) Get(ctx context.Context, module string, lookup string, value string) (map[string]any, error) {
	return a.store.Get(ctx, module, lookup, value)
}

func (a *storageAdapter) List(ctx context.Context, module string, opts runtime.ListOptions) ([]map[string]any, int64, error) {
	return a.store.List(ctx, module, storage.ListOptions{
		Limit:     opts.Limit,
		Offset:    opts.Offset,
		Filters:   opts.Filters,
		OrderBy:   opts.OrderBy,
		OrderDesc: opts.OrderDesc,
	})
}

func (a *storageAdapter) Update(ctx context.Context, module string, id string, data map[string]any) error {
	return a.store.Update(ctx, module, id, data)
}

func (a *storageAdapter) Delete(ctx context.Context, module string, id string) error {
	return a.store.Delete(ctx, module, id)
}
