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

	// Note: you'll need to set up sender identity verification
	// https://sendgrid.com/docs/for-developers/sending-email/sender-identity/
	var from string
	emailCmd.Flags().StringVar(&from, "from", "", "From email address")
	emailCmd.MarkFlagRequired("from")
	viper.BindPFlag("from", emailCmd.Flags().Lookup("from"))

	var email string
	emailCmd.Flags().StringVar(&email, "to", "", "Destination email address")
	emailCmd.MarkFlagRequired("to")
	viper.BindPFlag("to", emailCmd.Flags().Lookup("to"))

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

	out := ""
	for _, action := range actions {
		start := time.Date(now.Year()-1, now.Month(), 1, 0, 0, 0, 0, now.Location())
		end := start.AddDate(0, 1, 0)
		out += fmt.Sprintf("%s for %s %s:\n", action.GetName(), config.User, start.Format("2006-01"))
		topAlbumsOut, err := action.GetResults(config.DbPath, 20, start, end)
		if err != nil {
			return fmt.Errorf("sendEmail: %w", err)
		}
		out += topAlbumsOut + "\n\n"

		start = time.Date(now.Year()-2, now.Month(), 1, 0, 0, 0, 0, now.Location())
		end = start.AddDate(0, 1, 0)
		out += fmt.Sprintf("%s for %s %s:\n", action.GetName(), config.User, start.Format("2006-01"))
		topAlbumsOut, err = action.GetResults(config.DbPath, 20, start, end)
		if err != nil {
			return fmt.Errorf("sendEmail: %w", err)
		}
		out += topAlbumsOut + "\n\n"

		start = time.Date(now.Year()-3, now.Month(), 1, 0, 0, 0, 0, now.Location())
		end = start.AddDate(0, 1, 0)
		out += fmt.Sprintf("%s for %s %s:\n", action.GetName(), config.User, start.Format("2006-01"))
		topAlbumsOut, err = action.GetResults(config.DbPath, 20, start, end)
		if err != nil {
			return fmt.Errorf("sendEmail: %w", err)
		}
		out += topAlbumsOut + "\n\n"
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
		message := mail.NewSingleEmail(from, subject, to, out, fmt.Sprintf("<pre>%s</pre>", out))
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
