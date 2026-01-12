package cmd

import (
	"testing"
)

func TestUpdateCommand(t *testing.T) {
	if updateCmd == nil {
		t.Error("updateCmd is nil")
	}
	if updateCmd.Use != "update" {
		t.Errorf("expected use 'update', got %s", updateCmd.Use)
	}
}
