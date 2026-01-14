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
	"fmt"
	"net/smtp"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type SendEmailConfig struct {
	DbPath       string
	User         string
	From         string
	To           string
	ReportName   string
	Types        []string
	Params       []map[string]string
	DryRun       bool
	SMTPUsername string
	SMTPPassword string
	Start        time.Time
	End          time.Time
	NextRun      time.Time
}

var emailCmd = &cobra.Command{
	Use:   "email <address> <analysis_name...> [date] [date]",
	Short: "Sends an email report",
	Long: `Emails history to the specified user.
  <analysis_name> is one or more of: top-artists, top-albums, new-artists, new-albums, forgotten, top-n, taste-report.
  Optional date arguments can be provided at the end (e.g. '2023-01' or '2023-01 2023-06').
  If no dates are provided, defaults to the previous month.`,
	Args: cobra.MinimumNArgs(2),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if viper.GetString("from") == "" {
			return fmt.Errorf("required flag(s) \"from\" not set")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		to := args[0]
		rest := args[1:]

		// Try to parse dates from the end of the args
		var dateArgs []string
		// Check last arg
		if len(rest) > 0 {
			_, err := parseSingleDatestring(rest[len(rest)-1])
			if err == nil {
				dateArgs = []string{rest[len(rest)-1]}
				rest = rest[:len(rest)-1]

				// Check second to last
				if len(rest) > 0 {
					_, err := parseSingleDatestring(rest[len(rest)-1])
					if err == nil {
						dateArgs = append([]string{rest[len(rest)-1]}, dateArgs...)
						rest = rest[:len(rest)-1]
					}
				}
			}
		}

		analysisTypes := rest
		if len(analysisTypes) == 0 {
			fmt.Println("Error: No analysis types specified")
			os.Exit(1)
		}

		var start, end time.Time
		var err error
		if len(dateArgs) > 0 {
			start, end, err = parseDateRangeFromArgs(dateArgs)
			if err != nil {
				fmt.Printf("Error parsing dates: %v\n", err)
				os.Exit(1)
			}
		} else {
			// Default to last month
			now := time.Now()
			start = time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location())
			end = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		}

		params, _ := cmd.Flags().GetStringArray("params")
		
		if len(params) > 0 && len(params) != len(analysisTypes) {
			fmt.Printf("Error: Number of --params flags (%d) must match number of reports (%d), or be 0.\n", len(params), len(analysisTypes))
			os.Exit(1)
		}

		structuredParams := make([]map[string]string, len(analysisTypes))
		for i, v := range params {
			pMap := make(map[string]string)
			if v != "" {
				pairs := strings.Split(v, ",")
				for _, pair := range pairs {
					kv := strings.SplitN(pair, "=", 2)
					if len(kv) == 2 {
						pMap[kv[0]] = kv[1]
					}
				}
			}
			structuredParams[i] = pMap
		}

		config := SendEmailConfig{
			DbPath:       viper.GetString("database"),
			User:         viper.GetString("user"),
			From:         viper.GetString("from"),
			To:           to,
			ReportName:   viper.GetString("name"),
			Types:        analysisTypes,
			Params:       structuredParams,
			DryRun:       viper.GetBool("dryRun"),
			SMTPUsername: viper.GetString("smtp_username"),
			SMTPPassword: viper.GetString("smtp_password"),
			Start:        start,
			End:          end,
		}
		err = sendEmail(config)
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
	viper.BindPFlag("dryRun", emailCmd.Flags().Lookup("dry_run"))

	emailCmd.Flags().StringArray("params", nil, "Parameters for reports, matched by index (e.g. --params 'n=20')")
}

func sendEmail(config SendEmailConfig) error {
	actions := make([]Analyser, 0)
	for i, actionName := range config.Types {
		action, err := getActionFromName(actionName)
		if err != nil {
			return fmt.Errorf("Invalid analysis_name: %s", actionName)
		}

		if config.Params != nil && i < len(config.Params) {
			params := config.Params[i]
			if len(params) > 0 {
				if configurable, ok := action.(Configurable); ok {
					err := configurable.Configure(params)
					if err != nil {
						return fmt.Errorf("configuring %s (index %d): %w", actionName, i, err)
					}
				}
			}
		}

		actions = append(actions, action)
	}
	subject, out, err := generateEmailContent(config, actions)
	if err != nil {
		return err
	}

	if config.DryRun {
		fmt.Printf("Would have sent email: \nsubject: %s\n%s\n", subject, out)
	} else {
		if config.SMTPUsername == "" || config.SMTPPassword == "" {
			return fmt.Errorf("smtp_username and smtp_password must be set in order to send emails")
		}

		msg := "From: last-fm-tools <" + config.From + ">\r\n" +
			"To: " + config.To + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"MIME-Version: 1.0\r\n" +
			"Content-Type: text/html; charset=\"UTF-8\"\r\n" +
			"\r\n" +
			out

		auth := smtp.PlainAuth("", config.SMTPUsername, config.SMTPPassword, "smtp.gmail.com")
		err := smtp.SendMail("smtp.gmail.com:587", auth, config.From, []string{config.To}, []byte(msg))
		if err != nil {
			return fmt.Errorf("sendEmail: %w", err)
		}

		if len(config.ReportName) > 0 {
			db, err := createDatabase(config.DbPath)
			if err != nil {
				return fmt.Errorf("Recording last run: %w", err)
			}
			now := time.Now()
			
			if !config.NextRun.IsZero() {
				_, err = db.Exec("UPDATE Report SET sent = ?, next_run = ? WHERE user = ? AND name = ? AND email = ?", now, config.NextRun, config.User, config.ReportName, config.To)
			} else {
				_, err = db.Exec("UPDATE Report SET sent = ? WHERE user = ? AND name = ? AND email = ?", now, config.User, config.ReportName, config.To)
			}
			
			if err != nil {
				return fmt.Errorf("Recording last run: %w", err)
			}
			db.Close()
		}
	}

	return nil
}

func generateEmailContent(config SendEmailConfig, actions []Analyser) (subject string, body string, err error) {
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
		out += fmt.Sprintf("<h2>%s for %s %s to %s:</h2>\n", action.GetName(), config.User, config.Start.Format("2006-01-02"), config.End.Format("2006-01-02"))
		        		analysis, err := action.GetResults(config.DbPath, config.User, config.Start, config.End)
		        		if err == ErrSkipReport {
		        			fmt.Printf("Skipping report %q (check-sources): no issues detected.\n", config.ReportName)
		        			return "", "", nil
		        		}
		        		if err != nil {
		        
		            return "", "", fmt.Errorf("getting results for %s: %w", action.GetName(), err)
		        }
		
		        if analysis.BodyOverride != "" {			out += analysis.BodyOverride
		} else if len(analysis.results) <= 1 {
			// No listens found
			out += "<div>No listens found.</div>\n"
		} else {
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
		}
		out += fmt.Sprintf(`<div>%s</div>
		</div>`, analysis.summary)
	}

	subjectSuffix := ""
	if len(config.ReportName) > 0 {
		subjectSuffix = ": " + config.ReportName
	}
	// Subject line format: Listening report for <User> <Start> to <End> <Suffix>
	subject = fmt.Sprintf("Listening report for %s %s to %s%s", config.User, config.Start.Format("2006-01-02"), config.End.Format("2006-01-02"), subjectSuffix)

	return subject, out, nil
}

func getActionFromName(actionName string) (Analyser, error) {
	// Recreating map every time but it's fine. Pointers required for Configure.
	actionMap := map[string]Analyser{
		"top-artists":  &TopArtistsAnalyzer{Config: AnalyserConfig{20, 15}},
		"top-albums":   &TopAlbumsAnalyzer{Config: AnalyserConfig{20, 15}},
		"new-artists":  &NewArtistsAnalyzer{Config: AnalyserConfig{0, 5}},
		"new-albums":   &NewAlbumsAnalyzer{Config: AnalyserConfig{0, 5}},
		"forgotten":    &ForgottenAnalyzer{},
		"top-n":        &TopNAnalyzer{},
		"taste-report": &TasteReportAnalyzer{},
		"check-sources": &CheckSourcesAnalyzer{},
	}

	action, ok := actionMap[actionName]
	if !ok {
		return nil, fmt.Errorf("Invalid analysis_name: %s", actionName)
	}

	return action, nil
}
