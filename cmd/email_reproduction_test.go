package cmd

import (
	"testing"
	"time"
)

type MockSkipAnalyzer struct{}

func (m *MockSkipAnalyzer) GetName() string { return "Mock Skip" }
func (m *MockSkipAnalyzer) GetResults(dbPath string, user string, start, end time.Time) (Analysis, error) {
	return Analysis{}, ErrSkipReport
}

func TestGenerateEmailContent_WithErrSkipReport(t *testing.T) {
	db, dbPath := createTestDb(t)
	defer db.Close()

	user := "testuser"
	err := createUser(db, user)
	if err != nil {
		t.Fatalf("createUser: %w", err)
	}

	config := SendEmailConfig{
		DbPath:     dbPath,
		User:       user,
		ReportName: "Skip Report",
		Start:      time.Now(),
		End:        time.Now(),
	}

	actions := []Analyser{
		&MockSkipAnalyzer{},
	}

	subject, body, err := generateEmailContent(config, actions)

	if err != ErrNoDataToReport {
		t.Errorf("Expected ErrNoDataToReport, got: %v", err)
	}

	if subject != "" {
		t.Errorf("Expected empty subject, got: %s", subject)
	}

	if body != "" {
		t.Errorf("Expected empty body, got: %s", body)
	}
}
