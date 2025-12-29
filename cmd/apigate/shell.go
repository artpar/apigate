package main

import (
	"context"
	"os"

	"github.com/artpar/apigate/adapters/sqlite"
	"github.com/artpar/apigate/bootstrap"
	"github.com/artpar/apigate/core/channel/tty"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Start interactive shell",
	Long: `Start an interactive REPL for managing APIGate modules.

Examples:
  apigate shell

Interactive commands:
  modules              List available modules
  list <module>        List records
  get <module> <id>    Get record details
  create <module> ...  Create new record
  update <module> ...  Update record
  delete <module> <id> Delete record
  help                 Show help
  quit                 Exit shell`,
	RunE: runShell,
}

func init() {
	rootCmd.AddCommand(shellCmd)
}

func runShell(cmd *cobra.Command, args []string) error {
	// Setup logger
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	zerolog.SetGlobalLevel(zerolog.WarnLevel)

	// Get database path
	dsn := os.Getenv("APIGATE_DATABASE_DSN")
	if dsn == "" {
		dsn = "apigate.db"
	}

	// Check if database exists
	if _, err := os.Stat(dsn); os.IsNotExist(err) {
		return err
	}

	// Open database
	db, err := sqlite.Open(dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	// Create a dummy root command for CLI channel (won't be used)
	dummyRoot := &cobra.Command{Use: "shell"}

	// Create module runtime
	mr, err := bootstrap.NewModuleRuntime(db.DB, dummyRoot, logger, bootstrap.ModuleConfig{})
	if err != nil {
		return err
	}

	// Load modules
	ctx := context.Background()
	if err := mr.LoadModules(ctx, bootstrap.ModuleConfig{
		EmbeddedModules: bootstrap.CoreModules(),
	}); err != nil {
		return err
	}

	// Create and register TTY channel
	ttyChannel := tty.New(mr.Runtime)
	for _, mod := range mr.Modules() {
		derived, _ := mr.GetModule(mod.Name)
		ttyChannel.Register(derived)
	}

	// Run interactive shell
	return ttyChannel.Run(ctx)
}
