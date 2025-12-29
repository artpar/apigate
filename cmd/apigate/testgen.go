package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/artpar/apigate/bootstrap"
	"github.com/artpar/apigate/core/convention"
	"github.com/artpar/apigate/core/schema"
	"github.com/artpar/apigate/core/testgen"
	"github.com/spf13/cobra"
)

var testGenCmd = &cobra.Command{
	Use:   "test",
	Short: "Test generation and validation commands",
	Long: `Commands for generating and running tests from module schemas.

The test generator creates Go test files that validate:
  - Required field enforcement
  - Unique constraint enforcement
  - Enum value validation
  - Type validation (email, URL, int, etc.)
  - Constraint validation (min, max, pattern, etc.)
  - CRUD operations
  - Custom actions

Generated tests use the standard Go testing framework with testify assertions.`,
}

var testGenerateCmd = &cobra.Command{
	Use:   "generate [module]",
	Short: "Generate Go tests from module schemas",
	Long: `Generate Go test files from module schemas.

If no module is specified, generates tests for all modules.

Examples:
  # Generate tests for all modules
  apigate test generate --output ./generated_tests

  # Generate tests for a specific module
  apigate test generate user --output ./generated_tests

  # Generate to stdout
  apigate test generate user

  # Specify package name
  apigate test generate --package myapp_test --output ./tests`,
	RunE: runTestGenerate,
}

var (
	testOutputDir   string
	testPackageName string
	testListModules bool
)

func init() {
	rootCmd.AddCommand(testGenCmd)
	testGenCmd.AddCommand(testGenerateCmd)

	testGenerateCmd.Flags().StringVarP(&testOutputDir, "output", "o", "", "output directory for generated test files")
	testGenerateCmd.Flags().StringVarP(&testPackageName, "package", "p", "generated_test", "package name for generated tests")
	testGenerateCmd.Flags().BoolVarP(&testListModules, "list", "l", false, "list available modules without generating tests")
}

func runTestGenerate(cmd *cobra.Command, args []string) error {
	// Load module schemas (without full runtime)
	modules, err := loadModules()
	if err != nil {
		return fmt.Errorf("failed to load modules: %w", err)
	}

	// List modules only
	if testListModules {
		fmt.Println("Available modules:")
		for name := range modules {
			fmt.Printf("  - %s\n", name)
		}
		return nil
	}

	// Create generator
	gen := testgen.NewGenerator(modules)
	gen.SetPackageName(testPackageName)

	// Determine which modules to generate
	var targetModules []string
	if len(args) > 0 {
		// Specific module
		moduleName := args[0]
		if _, ok := modules[moduleName]; !ok {
			return fmt.Errorf("module %q not found. Use --list to see available modules", moduleName)
		}
		targetModules = []string{moduleName}
	} else {
		// All modules
		for name := range modules {
			targetModules = append(targetModules, name)
		}
	}

	// Generate tests
	for _, moduleName := range targetModules {
		code, err := gen.GenerateModule(moduleName)
		if err != nil {
			return fmt.Errorf("generating tests for %s: %w", moduleName, err)
		}

		if testOutputDir == "" {
			// Output to stdout
			fmt.Printf("// ============================================================\n")
			fmt.Printf("// Tests for module: %s\n", moduleName)
			fmt.Printf("// ============================================================\n\n")
			fmt.Println(string(code))
		} else {
			// Write to file
			if err := os.MkdirAll(testOutputDir, 0755); err != nil {
				return fmt.Errorf("creating output directory: %w", err)
			}

			filename := filepath.Join(testOutputDir, fmt.Sprintf("%s_test.go", toSnakeCase(moduleName)))
			if err := os.WriteFile(filename, code, 0644); err != nil {
				return fmt.Errorf("writing %s: %w", filename, err)
			}
			fmt.Printf("%s Generated: %s\n", checkMark, filename)
		}
	}

	if testOutputDir != "" {
		fmt.Printf("\n%s All tests generated in %s\n", checkMark, testOutputDir)
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Implement getTestRuntime() in the generated files")
		fmt.Println("  2. Run: go test ./...")
	}

	return nil
}

// loadModules loads module schemas for test generation (without full runtime).
func loadModules() (map[string]convention.Derived, error) {
	result := make(map[string]convention.Derived)

	// Load core embedded modules
	coreModules := bootstrap.CoreModules()
	for _, mod := range coreModules {
		derived := convention.Derive(mod)
		result[mod.Name] = derived
	}

	// Also try to load from modules directory if it exists
	modulesDir := bootstrap.CoreModulesDir()
	if _, err := os.Stat(modulesDir); err == nil {
		dirModules, err := schema.ParseDir(modulesDir)
		if err == nil {
			for _, mod := range dirModules {
				derived := convention.Derive(mod)
				result[mod.Name] = derived
			}
		}
	}

	return result, nil
}

// toSnakeCase converts CamelCase or kebab-case to snake_case.
func toSnakeCase(s string) string {
	// Handle kebab-case first
	s = strings.ReplaceAll(s, "-", "_")

	// Handle CamelCase
	var result []byte
	for i, c := range s {
		if i > 0 && c >= 'A' && c <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, byte(c))
	}
	return strings.ToLower(string(result))
}
