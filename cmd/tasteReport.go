package cmd

import (
	"fmt"
	"os"

	"github.com/ademuri/last-fm-tools/internal/analysis"
	"github.com/ademuri/last-fm-tools/internal/store"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var tasteReportCmd = &cobra.Command{
	Use:   "taste-report",
	Short: "Generates a comprehensive music taste report",
	Long:  `Analyzes your listening history to generate a detailed YAML report of your music taste, history, and drift.`,
	Run: func(cmd *cobra.Command, args []string) {
		err := runTasteReport()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating report: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(tasteReportCmd)
}

func runTasteReport() error {
	dbPath := viper.GetString("database")
	user := viper.GetString("user")

	// Check if DB exists by trying to open it via Store
	// Store.New will create tables if they don't exist, which is fine.
	// But we might want to check if there is data?
	// The report generation will fail or return empty if no data.
	
db, err := store.New(dbPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

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