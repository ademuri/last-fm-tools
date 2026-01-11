package cmd

import (
	"fmt"
	"os"

	"github.com/ademuri/last-fm-tools/internal/analysis"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generates a comprehensive music taste report",
	Long:  `Analyzes your listening history to generate a detailed YAML report of your music taste, history, and drift.`,
	Run: func(cmd *cobra.Command, args []string) {
		err := runReport()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating report: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(reportCmd)
}

func runReport() error {
	dbPath := viper.GetString("database")
	user := viper.GetString("user")

	db, err := openDb(dbPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	// Check if DB has data
	exists, err := dbExists(db)
	if err != nil {
		return fmt.Errorf("checking db: %w", err)
	}
	if !exists {
		return fmt.Errorf("database empty or missing. Run 'update' first")
	}

	report, err := analysis.GenerateReport(db, user)
	if err != nil {
		return fmt.Errorf("analyzing data: %w", err)
	}

	encoder := yaml.NewEncoder(os.Stdout)
	encoder.SetIndent(2)
	err = encoder.Encode(report)
	if err != nil {
		return fmt.Errorf("encoding report: %w", err)
	}

	return nil
}
