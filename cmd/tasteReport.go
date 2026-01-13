package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

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

type TasteReportAnalyzer struct {
}

func (t *TasteReportAnalyzer) Configure(params map[string]string) error {
	// No params for now
	return nil
}

func (t *TasteReportAnalyzer) GetName() string {
	return "Music Taste Profile"
}

func (t *TasteReportAnalyzer) GetResults(dbPath string, user string, start time.Time, end time.Time) (Analysis, error) {
	var a Analysis
	db, err := store.New(dbPath)
	if err != nil {
		return a, fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	report, err := analysis.GenerateReport(db, user)
	if err != nil {
		return a, fmt.Errorf("generating report: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<p><strong>Analysis Date:</strong> %s</p>", report.Metadata.GeneratedDate))
	sb.WriteString(fmt.Sprintf("<p><strong>Current Period:</strong> %s</p>", report.Metadata.CurrentPeriod))
	
	// Top Artists
	sb.WriteString("<h3>Current Top Artists</h3>")
	sb.WriteString("<ul>")
	for i, artist := range report.CurrentTaste.TopArtists {
		if i >= 10 { break }
		sb.WriteString(fmt.Sprintf("<li><strong>%s</strong> (%d scrobbles)", artist.Name, artist.Scrobbles))
		if len(artist.PrimaryTags) > 0 {
			sb.WriteString(fmt.Sprintf(" [%s]", strings.Join(artist.PrimaryTags, ", ")))
		}
		sb.WriteString("</li>")
	}
	sb.WriteString("</ul>")

	// Taste Drift
	sb.WriteString("<h3>Taste Drift</h3>")
	if len(report.TasteDrift.EmergedTags) > 0 {
		sb.WriteString("<p><strong>New Interests:</strong> ")
		var tags []string
		for _, t := range report.TasteDrift.EmergedTags {
			tags = append(tags, t.Tag)
		}
		sb.WriteString(strings.Join(tags, ", "))
		sb.WriteString("</p>")
	}
	if len(report.TasteDrift.DeclinedTags) > 0 {
		sb.WriteString("<p><strong>Fading Interests:</strong> ")
		var tags []string
		for _, t := range report.TasteDrift.DeclinedTags {
			tags = append(tags, t.Tag)
		}
		sb.WriteString(strings.Join(tags, ", "))
		sb.WriteString("</p>")
	}

	a.BodyOverride = sb.String()
	return a, nil
}