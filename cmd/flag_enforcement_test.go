package cmd

import (
	"testing"

	"github.com/spf13/viper"
)

func TestAddReportRequiresUser(t *testing.T) {
	// Reset viper
	viper.Reset()
	addReportCmd.Flags().Set("dest", "test@example.com")
	addReportCmd.Flags().Set("name", "test report")
	
	// Ensure user is empty
	viper.Set("user", "")

	err := addReportCmd.PreRunE(addReportCmd, []string{"top-artists"})
	if err == nil {
		t.Error("Expected error when user is missing, got nil")
	} else if err.Error() != "required flag(s) \"user\" not set" {
		t.Errorf("Expected 'required flag(s) \"user\" not set', got %v", err)
	}

	// Set user and check success
	viper.Set("user", "testuser")
	err = addReportCmd.PreRunE(addReportCmd, []string{"top-artists"})
	if err != nil {
		t.Errorf("Expected nil when user is set, got %v", err)
	}
}

func TestDeleteReportRequiresUser(t *testing.T) {
	// Reset viper
	viper.Reset()
	
	// Ensure user is empty
	viper.Set("user", "")

	err := deleteReportCmd.PreRunE(deleteReportCmd, []string{})
	if err == nil {
		t.Error("Expected error when user is missing, got nil")
	} else if err.Error() != "required flag(s) \"user\" not set" {
		t.Errorf("Expected 'required flag(s) \"user\" not set', got %v", err)
	}

	// Set user and check success
	viper.Set("user", "testuser")
	err = deleteReportCmd.PreRunE(deleteReportCmd, []string{})
	if err != nil {
		t.Errorf("Expected nil when user is set, got %v", err)
	}
}
