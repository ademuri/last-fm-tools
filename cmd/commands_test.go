package cmd

import (
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func IntegrationTestAddReport(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	
	// Make sure unneeded flags are not set
	viper.Set("api_key", "")
	viper.Set("secret", "")

	// Reset args
	rootCmd.SetArgs([]string{
		"add-report",
		"top-artists",
		"--dest", "test@example.com",
		"--name", "testreport",
		"--database", dbPath,
		"--user", "testuser",
	})

	// Execute
	err := rootCmd.Execute()

	// Assert
	if err != nil {
		t.Fatalf("add-report failed: %v", err)
	}
}

func IntegrationTestUpdate(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Set required flags
	viper.Set("api_key", "test-api-key")
	viper.Set("secret", "test-secret")

	// Reset args
	rootCmd.SetArgs([]string{
		"update",
		"--database", dbPath,
		"--user", "testuser",
	})

	// Execute
	err := rootCmd.Execute()

	// Assert
	if err != nil {
		t.Fatal("update failed: %v", err)
	}
}
