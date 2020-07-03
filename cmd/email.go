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
	"os"
	"strconv"
	"time"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var emailCmd = &cobra.Command{
	Use:   "email <address> <analysis_name...>",
	Short: "Sends an email report",
	Long: `Emails history to the specified user.
  <analysis_name> is one or more of: top-artists, top-albums, new-artists, new-albums`,
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		err := sendEmail(viper.GetString("database"), viper.GetString("from"), viper.GetBool("dry_run"), viper.GetString("name"), viper.GetString("run_day"), args)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(emailCmd)

	// Note: you'll need to set up sender identity verification
	// https://sendgrid.com/docs/for-developers/sending-email/sender-identity/
	var from string
	emailCmd.Flags().StringVar(&from, "from", "", "From email address")
	viper.BindPFlag("from", emailCmd.Flags().Lookup("from"))

	var dryRun bool
	emailCmd.Flags().BoolVarP(&dryRun, "dry_run", "n", false, "When true, just print instead of emailing")
	viper.BindPFlag("dry_run", emailCmd.Flags().Lookup("dry_run"))

	var reportName string
	emailCmd.Flags().StringVar(&reportName, "name", "", "Report name - included in the email title, and used for periodically sending")
	viper.BindPFlag("name", emailCmd.Flags().Lookup("name"))

	var runDay string
	emailCmd.Flags().StringVar(&runDay, "run_day", "", "Which day of the month to run this report on")
	viper.BindPFlag("run_day", emailCmd.Flags().Lookup("run_day"))
}

func sendEmail(dbPath string, fromAddress string, dryRun bool, reportName string, runDay string, args []string) error {
	if len(runDay) > 0 && len(reportName) == 0 {
		return fmt.Errorf("When specifying --run_day, you must also specify --name")
	}

	now := time.Now()
	user := viper.GetString("user")
	toAddress := args[0]

	if len(runDay) > 0 {
		runDayNumber, err := strconv.Atoi(runDay)
		if err != nil {
			return fmt.Errorf("Parsing runDay: %w", err)
		}
		if runDayNumber < 0 || runDayNumber > 31 {
			return fmt.Errorf("runDay out of range: %d", runDayNumber)
		}

		db, err := createDatabase(dbPath)
		if err != nil {
			return fmt.Errorf("Checking last run: %w", err)
		}
		sentQuery, err := db.Query("SELECT sent FROM Report WHERE user = ? AND name = ? AND email = ?", user, reportName, toAddress)
		if sentQuery.Next() {
			var sent time.Time
			sentQuery.Scan(&sent)
			toSendThisMonth := time.Date(now.Year(), now.Month(), runDayNumber, 0, 0, 0, 0, now.Location())
			toSendLastMonth := time.Date(now.Year(), now.Month()-1, runDayNumber, 0, 0, 0, 0, now.Location())
			if sent.After(toSendThisMonth) {
				fmt.Printf("Report was already sent this month on %s, not sending.\n", sent.Format("2006-01-02"))
				return nil
			}
			if now.After(toSendThisMonth) && sent.After(toSendLastMonth) {
				fmt.Printf("Report was already sent for last month on %s, not sending.\n", sent.Format("2006-01-02"))
				return nil
			}
		} else {
			fmt.Println("No previous runs, running now...")
		}

		sentQuery.Close()
		db.Close()
	}

	actionMap := map[string]Analyser{
		"top-artists": TopArtistsAnalyzer{},
		"top-albums":  TopAlbumsAnalyzer{},
		"new-artists": NewArtistsAnalyzer{},
		"new-albums":  NewAlbumsAnalyzer{},
	}
	actions := make([]Analyser, 0)
	for _, actionName := range args[1:] {
		action, ok := actionMap[actionName]
		if !ok {
			return fmt.Errorf("Invalid analysis_name: %s", actionName)
		}
		actions = append(actions, action)
	}

	out := ""
	for _, action := range actions {
		start := time.Date(now.Year()-1, now.Month(), 1, 0, 0, 0, 0, now.Location())
		end := start.AddDate(0, 1, 0)
		out += fmt.Sprintf("%s for %s:\n", action.GetName(), start.Format("2006-01"))
		topAlbumsOut, err := action.GetResults(dbPath, 20, start, end)
		if err != nil {
			return fmt.Errorf("sendEmail: %w", err)
		}
		out += topAlbumsOut + "\n\n"

		start = time.Date(now.Year()-2, now.Month(), 1, 0, 0, 0, 0, now.Location())
		end = start.AddDate(0, 1, 0)
		out += fmt.Sprintf("%s for %s:\n", action.GetName(), start.Format("2006-01"))
		topAlbumsOut, err = action.GetResults(dbPath, 20, start, end)
		if err != nil {
			return fmt.Errorf("sendEmail: %w", err)
		}
		out += topAlbumsOut + "\n\n"

		start = time.Date(now.Year()-3, now.Month(), 1, 0, 0, 0, 0, now.Location())
		end = start.AddDate(0, 1, 0)
		out += fmt.Sprintf("%s for %s:\n", action.GetName(), start.Format("2006-01"))
		topAlbumsOut, err = action.GetResults(dbPath, 20, start, end)
		if err != nil {
			return fmt.Errorf("sendEmail: %w", err)
		}
		out += topAlbumsOut + "\n\n"
	}

	if dryRun {
		fmt.Printf("Would have sent email: \n%s\n", out)
	} else {
		from := mail.NewEmail("last-fm-tools", fromAddress)
		subjectSuffix := ""
		if len(reportName) > 0 {
			subjectSuffix = ": " + reportName
		}
		subject := fmt.Sprintf("Listening report for %s%s", now.Format("2006-01"), subjectSuffix)

		to := mail.NewEmail(toAddress, toAddress)
		message := mail.NewSingleEmail(from, subject, to, out, fmt.Sprintf("<pre>%s</pre>", out))
		client := sendgrid.NewSendClient(viper.GetString("sendgrid_api_key"))
		_, err := client.Send(message)
		if err != nil {
			return fmt.Errorf("sendEmail: %w", err)
		}
	}

	if len(reportName) > 0 {
		db, err := createDatabase(dbPath)
		if err != nil {
			return fmt.Errorf("Recording last run: %w", err)
		}
		_, err = db.Exec("REPLACE INTO Report(user, name, email, sent) VALUES (?, ?, ?, ?)", user, reportName, toAddress, now)
		if err != nil {
			return fmt.Errorf("Recording last run: %w", err)
		}
		db.Close()
	}

	return nil
}
