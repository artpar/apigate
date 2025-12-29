// Package tty provides an interactive terminal channel for modules.
// It creates a REPL-style interface for managing module data.
package tty

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	goruntime "runtime"
	"strings"
	"time"

	"github.com/artpar/apigate/core/convention"
	"github.com/artpar/apigate/core/runtime"
	"github.com/artpar/apigate/core/schema"
)

// Channel implements the TTY channel for interactive terminal sessions.
type Channel struct {
	runtime  *runtime.Runtime
	modules  map[string]convention.Derived
	prompt   string
	running  bool
	showStats bool // Show execution stats after each command
}

// ExecStats holds execution statistics.
type ExecStats struct {
	Duration   time.Duration
	MemAlloc   uint64 // bytes allocated
	MemTotal   uint64 // total memory from system
	NumGC      uint32 // number of GCs
}

// New creates a new TTY channel.
func New(rt *runtime.Runtime) *Channel {
	return &Channel{
		runtime:   rt,
		modules:   make(map[string]convention.Derived),
		prompt:    "apigate> ",
		showStats: true, // Enable stats by default
	}
}

// captureStats captures current memory stats.
func captureStats() goruntime.MemStats {
	var m goruntime.MemStats
	goruntime.ReadMemStats(&m)
	return m
}

// formatBytes formats bytes as human readable.
func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// printStats prints execution statistics.
func printStats(duration time.Duration, before, after goruntime.MemStats) {
	memUsed := int64(after.Alloc) - int64(before.Alloc)
	if memUsed < 0 {
		memUsed = 0 // GC happened
	}
	gcRuns := after.NumGC - before.NumGC

	fmt.Printf("\033[90m") // dim gray
	fmt.Printf("  ⏱ %v", duration.Round(time.Microsecond))
	fmt.Printf("  📦 %s", formatBytes(uint64(memUsed)))
	fmt.Printf("  🗄 %s", formatBytes(after.Sys))
	if gcRuns > 0 {
		fmt.Printf("  ♻ %d GC", gcRuns)
	}
	fmt.Printf("\033[0m\n") // reset
}

// Name returns the channel name.
func (c *Channel) Name() string {
	return "tty"
}

// Register registers a module with the TTY channel.
func (c *Channel) Register(mod convention.Derived) error {
	// TTY channel registers all modules (interactive shell mode)
	c.modules[mod.Source.Name] = mod
	return nil
}

// Start starts the TTY channel (no-op, use Run for interactive mode).
func (c *Channel) Start(ctx context.Context) error {
	return nil
}

// Stop stops the TTY channel.
func (c *Channel) Stop(ctx context.Context) error {
	c.running = false
	return nil
}

// Run starts the interactive REPL.
func (c *Channel) Run(ctx context.Context) error {
	c.running = true
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("APIGate Interactive Shell")
	fmt.Println("Type 'help' for available commands, 'quit' to exit")
	fmt.Println()

	for c.running {
		fmt.Print(c.prompt)
		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Capture stats before execution
		before := captureStats()
		start := time.Now()

		err := c.execute(ctx, line)

		// Capture stats after execution
		duration := time.Since(start)
		after := captureStats()

		if err != nil {
			fmt.Printf("Error: %v\n", err)
		}

		// Show execution stats
		if c.showStats && line != "quit" && line != "exit" && line != "q" {
			printStats(duration, before, after)
		}
		// Note: Analytics is automatically recorded by runtime.Execute()
	}

	return scanner.Err()
}

// execute parses and executes a command line.
func (c *Channel) execute(ctx context.Context, line string) error {
	parts := parseArgs(line)
	if len(parts) == 0 {
		return nil
	}

	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "quit", "exit", "q":
		c.running = false
		fmt.Println("Goodbye!")
		return nil

	case "help", "h", "?":
		return c.showHelp(args)

	case "list", "ls":
		return c.listRecords(ctx, args)

	case "get", "show":
		return c.getRecord(ctx, args)

	case "create", "new", "add":
		return c.createRecord(ctx, args)

	case "update", "set":
		return c.updateRecord(ctx, args)

	case "delete", "rm", "remove":
		return c.deleteRecord(ctx, args)

	case "modules":
		return c.listModules()

	case "stats":
		c.showStats = !c.showStats
		if c.showStats {
			fmt.Println("Stats display enabled")
		} else {
			fmt.Println("Stats display disabled")
		}
		return nil

	default:
		// Check if it's a module name followed by action
		if mod, ok := c.modules[cmd]; ok {
			return c.executeModuleCommand(ctx, mod, args)
		}
		return fmt.Errorf("unknown command: %s (try 'help')", cmd)
	}
}

// showHelp displays help information.
func (c *Channel) showHelp(args []string) error {
	if len(args) > 0 {
		if mod, ok := c.modules[args[0]]; ok {
			fmt.Printf("\n%s commands:\n", mod.Source.Name)
			fmt.Printf("  list                    List all %s\n", mod.Plural)
			fmt.Printf("  get <id-or-name>        Show %s details\n", mod.Source.Name)
			fmt.Printf("  create <field=value>... Create new %s\n", mod.Source.Name)
			fmt.Printf("  update <id> <field=val> Update %s\n", mod.Source.Name)
			fmt.Printf("  delete <id-or-name>     Delete %s\n", mod.Source.Name)
			for _, action := range mod.Actions {
				if action.Type == schema.ActionTypeCustom {
					fmt.Printf("  %s <id-or-name>       %s\n", action.Name, action.Description)
				}
			}
			fmt.Println()
			return nil
		}
	}

	fmt.Println("\nAvailable commands:")
	fmt.Println("  modules                 List available modules")
	fmt.Println("  list <module>           List records in module")
	fmt.Println("  get <module> <id>       Get record by ID or name")
	fmt.Println("  create <module> ...     Create new record")
	fmt.Println("  update <module> ...     Update record")
	fmt.Println("  delete <module> <id>    Delete record")
	fmt.Println("  <module> <action> ...   Execute module action")
	fmt.Println("  help [module]           Show help")
	fmt.Println("  quit                    Exit shell")
	fmt.Println("\nAvailable modules:")
	for name := range c.modules {
		fmt.Printf("  %s\n", name)
	}
	fmt.Println()
	return nil
}

// listModules lists available modules.
func (c *Channel) listModules() error {
	fmt.Println("\nAvailable modules:")
	for name, mod := range c.modules {
		fmt.Printf("  %-15s %d fields, %d actions\n", name, len(mod.Fields), len(mod.Actions))
	}
	fmt.Println()
	return nil
}

// listRecords lists records from a module.
func (c *Channel) listRecords(ctx context.Context, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: list <module>")
	}

	mod, ok := c.modules[args[0]]
	if !ok {
		return fmt.Errorf("unknown module: %s", args[0])
	}

	result, err := c.runtime.Execute(ctx, mod.Source.Name, "list", runtime.ActionInput{
		Channel: "tty",
	})
	if err != nil {
		return err
	}

	if len(result.List) == 0 {
		fmt.Printf("No %s found.\n", mod.Plural)
		return nil
	}

	// Print as simple table
	fmt.Printf("\n%s (%d):\n", mod.Plural, len(result.List))
	for _, record := range result.List {
		id := record["id"]
		name := record["name"]
		if name == nil {
			name = record["email"]
		}
		if name == nil {
			name = record["key"]
		}
		if name != nil {
			fmt.Printf("  %v  %v\n", id, name)
		} else {
			fmt.Printf("  %v\n", id)
		}
	}
	fmt.Println()
	return nil
}

// getRecord gets a single record.
func (c *Channel) getRecord(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: get <module> <id-or-name>")
	}

	mod, ok := c.modules[args[0]]
	if !ok {
		return fmt.Errorf("unknown module: %s", args[0])
	}

	result, err := c.runtime.Execute(ctx, mod.Source.Name, "get", runtime.ActionInput{
		Lookup:  args[1],
		Channel: "tty",
	})
	if err != nil {
		return err
	}

	fmt.Println()
	for _, f := range mod.Fields {
		if !f.Internal && f.Type != schema.FieldTypeSecret {
			val := result.Data[f.Name]
			if val != nil {
				fmt.Printf("  %-20s %v\n", f.Name+":", formatValue(val))
			}
		}
	}
	fmt.Println()
	return nil
}

// createRecord creates a new record.
func (c *Channel) createRecord(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: create <module> <field=value> ...")
	}

	mod, ok := c.modules[args[0]]
	if !ok {
		return fmt.Errorf("unknown module: %s", args[0])
	}

	data := parseKeyValues(args[1:])

	result, err := c.runtime.Execute(ctx, mod.Source.Name, "create", runtime.ActionInput{
		Data:    data,
		Channel: "tty",
	})
	if err != nil {
		return err
	}

	fmt.Printf("Created %s: %s\n", mod.Source.Name, result.ID)
	return nil
}

// updateRecord updates a record.
func (c *Channel) updateRecord(ctx context.Context, args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("usage: update <module> <id-or-name> <field=value> ...")
	}

	mod, ok := c.modules[args[0]]
	if !ok {
		return fmt.Errorf("unknown module: %s", args[0])
	}

	data := parseKeyValues(args[2:])

	_, err := c.runtime.Execute(ctx, mod.Source.Name, "update", runtime.ActionInput{
		Lookup:  args[1],
		Data:    data,
		Channel: "tty",
	})
	if err != nil {
		return err
	}

	fmt.Printf("Updated %s: %s\n", mod.Source.Name, args[1])
	return nil
}

// deleteRecord deletes a record.
func (c *Channel) deleteRecord(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: delete <module> <id-or-name>")
	}

	mod, ok := c.modules[args[0]]
	if !ok {
		return fmt.Errorf("unknown module: %s", args[0])
	}

	_, err := c.runtime.Execute(ctx, mod.Source.Name, "delete", runtime.ActionInput{
		Lookup:  args[1],
		Channel: "tty",
	})
	if err != nil {
		return err
	}

	fmt.Printf("Deleted %s: %s\n", mod.Source.Name, args[1])
	return nil
}

// executeModuleCommand executes a module-specific command.
func (c *Channel) executeModuleCommand(ctx context.Context, mod convention.Derived, args []string) error {
	if len(args) < 1 {
		return c.showHelp([]string{mod.Source.Name})
	}

	action := args[0]
	actionArgs := args[1:]

	switch action {
	case "list", "ls":
		return c.listRecords(ctx, []string{mod.Source.Name})
	case "get", "show":
		if len(actionArgs) < 1 {
			return fmt.Errorf("usage: %s get <id-or-name>", mod.Source.Name)
		}
		return c.getRecord(ctx, []string{mod.Source.Name, actionArgs[0]})
	case "create", "new":
		return c.createRecord(ctx, append([]string{mod.Source.Name}, actionArgs...))
	case "update", "set":
		return c.updateRecord(ctx, append([]string{mod.Source.Name}, actionArgs...))
	case "delete", "rm":
		if len(actionArgs) < 1 {
			return fmt.Errorf("usage: %s delete <id-or-name>", mod.Source.Name)
		}
		return c.deleteRecord(ctx, []string{mod.Source.Name, actionArgs[0]})
	default:
		// Try custom action
		for _, act := range mod.Actions {
			if act.Name == action && act.Type == schema.ActionTypeCustom {
				if len(actionArgs) < 1 {
					return fmt.Errorf("usage: %s %s <id-or-name>", mod.Source.Name, action)
				}
				result, err := c.runtime.Execute(ctx, mod.Source.Name, action, runtime.ActionInput{
					Lookup:  actionArgs[0],
					Channel: "tty",
				})
				if err != nil {
					return err
				}
				fmt.Printf("%s completed: %s\n", action, result.ID)
				return nil
			}
		}
		return fmt.Errorf("unknown action: %s (try 'help %s')", action, mod.Source.Name)
	}
}

// parseArgs parses a command line respecting quoted strings.
func parseArgs(line string) []string {
	var args []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for _, r := range line {
		switch {
		case r == '"' || r == '\'':
			if inQuote && r == quoteChar {
				inQuote = false
				quoteChar = 0
			} else if !inQuote {
				inQuote = true
				quoteChar = r
			} else {
				current.WriteRune(r)
			}
		case r == ' ' || r == '\t':
			if inQuote {
				current.WriteRune(r)
			} else if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}

// parseKeyValues parses "key=value" arguments into a map.
func parseKeyValues(args []string) map[string]any {
	data := make(map[string]any)
	for _, arg := range args {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) == 2 {
			data[parts[0]] = parts[1]
		}
	}
	return data
}

// formatValue formats a value for display.
func formatValue(val any) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	case bool:
		if v {
			return "yes"
		}
		return "no"
	case []byte:
		return "[binary]"
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}
