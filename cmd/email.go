/*
Copyright 2020 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type SendEmailConfig struct {
	DbPath     string
	User       string
	From       string
	To         string
	ReportName string
	Types      []string
	DryRun     bool
}

var emailCmd = &cobra.Command{
	Use:   "email <address> <analysis_name...>",
	Short: "Sends an email report",
	Long: `Emails history to the specified user.
  <analysis_name> is one or more of: top-artists, top-albums, new-artists, new-albums`,
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		config := SendEmailConfig{
			DbPath:     viper.GetString("database"),
			User:       viper.GetString("user"),
			From:       viper.GetString("from"),
			To:         args[0],
			ReportName: viper.GetString("name"),
			Types:      args[1:],
			DryRun:     viper.GetBool("dry_run"),
		}
		err := sendEmail(config)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(emailCmd)

	var dryRun bool
	emailCmd.Flags().BoolVarP(&dryRun, "dry_run", "n", false, "When true, just print instead of emailing")
	viper.BindPFlag("dry_run", emailCmd.Flags().Lookup("dry_run"))
}

func sendEmail(config SendEmailConfig) error {
	now := time.Now()

	actions := make([]Analyser, 0)
	for _, actionName := range config.Types {
		action, err := getActionFromName(actionName)
		if err != nil {
			return fmt.Errorf("Invalid analysis_name: %s", actionName)
		}
		actions = append(actions, action)
	}

	db, err := createDatabase(config.DbPath)
	years, err := getNumYearsOfListeningData(db, config.User)
	if err != nil {
		return fmt.Errorf("getNumYearsOfListeningData(): %w")
	}
	err = db.Close()
	if err != nil {
		return fmt.Errorf("closing database: %w")
	}

	out := `
<html>
  <head>
<style>
td {
  padding: 0.1em 0.2em;
}
table, th, td {
  border: 1px solid black;
  border-collapse: collapse;
}
</style>
  </head>
  <body>
`
	for _, action := range actions {
		out += `
		<div>
`
		for year := 1; year < years; year++ {
			start := time.Date(now.Year()-year, now.Month(), 1, 0, 0, 0, 0, now.Location())
			end := start.AddDate(0, 1, 0)
			out += fmt.Sprintf("<h2>%s for %s %s:</h2>\n", action.GetName(), config.User, start.Format("2006-01"))
			analysis, err := action.GetResults(config.DbPath, config.User, 20, start, end)
			if err != nil {
				return fmt.Errorf("sendEmail: %w", err)
			}

			out += `
			<table>
				<thead>
					<tr>
`
			for _, header := range analysis.results[0] {
				out += fmt.Sprintf("<th>%s</th>", header)
			}
			out += `				</tr>
			</thead>`

			for _, row := range analysis.results[1:] {
				out += "<tr>\n"
				for _, column := range row {
					out += fmt.Sprintf("<td>%s</td>\n", column)
				}
				out += "</tr>\n"

			}
			out += `
				</tbody>
			</table>
`
			out += fmt.Sprintf(`<div>%s</div>
		</div>`, analysis.summary)
		}
	}

	subjectSuffix := ""
	if len(config.ReportName) > 0 {
		subjectSuffix = ": " + config.ReportName
	}
	subject := fmt.Sprintf("Listening report for %s %s%s", config.User, now.Format("2006-01"), subjectSuffix)

	if config.DryRun {
		fmt.Printf("Would have sent email: \nsubject: %s\n%s\n", subject, out)
	} else {
		from := mail.NewEmail("last-fm-tools", config.From)

		to := mail.NewEmail(config.To, config.To)
		message := mail.NewSingleEmail(from, subject, to, out, fmt.Sprintf("%s", out))
		client := sendgrid.NewSendClient(viper.GetString("sendgrid_api_key"))
		_, err := client.Send(message)
		if err != nil {
			return fmt.Errorf("sendEmail: %w", err)
		}
	}

	if len(config.ReportName) > 0 {
		db, err := createDatabase(config.DbPath)
		if err != nil {
			return fmt.Errorf("Recording last run: %w", err)
		}
		_, err = db.Exec("UPDATE Report SET sent = ? WHERE user = ? AND name = ? AND email = ?", now, config.User, config.ReportName, config.To)
		if err != nil {
			return fmt.Errorf("Recording last run: %w", err)
		}
		db.Close()
	}

	return nil
}

func getActionFromName(actionName string) (Analyser, error) {
	actionMap := map[string]Analyser{
		"top-artists": TopArtistsAnalyzer{},
		"top-albums":  TopAlbumsAnalyzer{},
		"new-artists": NewArtistsAnalyzer{},
		"new-albums":  NewAlbumsAnalyzer{},
	}

	action, ok := actionMap[actionName]
	if !ok {
		return nil, fmt.Errorf("Invalid analysis_name: %s", actionName)
	}

	return action, nil
}

func getNumYearsOfListeningData(db *sql.DB, user string) (int, error) {
	query, err := db.Query("SELECT date FROM Listen WHERE user = ? ORDER BY date ASC LIMIT 1;", user)
	if err != nil {
		err = fmt.Errorf("query(%q): %w", user, err)
		return 0, err
	}
	defer query.Close()

	if !query.Next() {
		return 0, fmt.Errorf("No listens found for user %q", user)
	}

	var oldest time.Time
	query.Scan(&oldest)

	now := time.Now()
	years := now.Year() - oldest.Year()
	if now.Month() > oldest.Month() || (now.Month() == oldest.Month() && now.Day() >= oldest.Day()) {
		years++
	}
	return years, nil
}
