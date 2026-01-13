/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

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
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type SendReportsConfig struct {
	DbPath       string
	From         string
	DryRun       bool
	SMTPUsername string
	SMTPPassword string
	Force        bool
	User         string
}

var sendReportsCmd = &cobra.Command{
	Use:   "send-reports",
	Short: "Update the database and send email reports.",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		dryRun, _ := cmd.Flags().GetBool("dry_run")
		force, _ := cmd.Flags().GetBool("force")
		config := SendReportsConfig{
			DbPath:       viper.GetString("database"),
			From:         viper.GetString("from"),
			DryRun:       dryRun,
			SMTPUsername: viper.GetString("smtp_username"),
			SMTPPassword: viper.GetString("smtp_password"),
			Force:        force,
			User:         viper.GetString("user"),
		}
		err := sendReports(config)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(sendReportsCmd)

	var dryRun bool
	sendReportsCmd.Flags().BoolVarP(&dryRun, "dry_run", "n", false, "When true, just print instead of emailing")

	var force bool
	sendReportsCmd.Flags().BoolVarP(&force, "force", "f", false, "When true, send reports even if they've already been sent")
}

func sendReports(config SendReportsConfig) error {
	db, err := createDatabase(config.DbPath)
	if err != nil {
		return err
	}
	reports, err := db.Query("SELECT name, user, email, sent, run_day, types, params FROM Report")
	if err != nil {
		return fmt.Errorf("Querying reports: %w", err)
	}

	emailConfigs := make([]SendEmailConfig, 0)
	for reports.Next() {
		var (
			sentOrNull sql.NullTime
			runDay     int
			types      string
			params     sql.NullString
		)

		now := time.Now()
		emailConfig := SendEmailConfig{
			DbPath:       config.DbPath,
			From:         config.From,
			DryRun:       config.DryRun,
			SMTPUsername: config.SMTPUsername,
			SMTPPassword: config.SMTPPassword,
		}
		err = reports.Scan(&emailConfig.ReportName, &emailConfig.User, &emailConfig.To, &sentOrNull, &runDay, &types, &params)
		if err != nil {
			return fmt.Errorf("Getting report params: %w", err)
		}

		if config.User != "" && !strings.EqualFold(config.User, emailConfig.User) {
			continue
		}

		var sent time.Time
		if sentOrNull.Valid {
			sent = sentOrNull.Time
		}

		if params.Valid && params.String != "" {
			var p map[string]map[string]string
			if err := json.Unmarshal([]byte(params.String), &p); err != nil {
				fmt.Printf("Warning: failed to unmarshal params for report %s: %v\n", emailConfig.ReportName, err)
			} else {
				emailConfig.Params = p
			}
		}

		emailConfig.Types = strings.Split(types, ",")
		toSendThisMonth := time.Date(now.Year(), now.Month(), runDay, 0, 0, 0, 0, now.Location())
		toSendLastMonth := time.Date(now.Year(), now.Month()-1, runDay, 0, 0, 0, 0, now.Location())

		// Report covers the previous month
		emailConfig.Start = time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location())
		emailConfig.End = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

		if !config.Force {
			if sent.After(toSendThisMonth) {
				fmt.Printf("Report (%q, %q) was already sent this month on %s, not sending.\n", emailConfig.User, emailConfig.ReportName, sent.Format("2006-01-02"))
				continue
			}
			if now.Before(toSendThisMonth) && sent.After(toSendLastMonth) {
				fmt.Printf("Report (%q, %q) was already sent for last month on %s, not sending.\n", emailConfig.User, emailConfig.ReportName, sent.Format("2006-01-02"))
				continue
			}
		}

		emailConfigs = append(emailConfigs, emailConfig)
	}
	reports.Close()

	errOccurred := false
	for _, emailConfig := range emailConfigs {
		if !config.DryRun {
			updateConfig := UpdateConfig{
				DbPath: config.DbPath,
				User:   emailConfig.User,
			}

			err = updateDatabase(updateConfig)
			if err != nil {
				errOccurred = true
				fmt.Printf("updateDatabase(%q): %w", emailConfig.User, err)
			}
		}

		fmt.Printf("Sending report (%q, %q)\n", emailConfig.User, emailConfig.ReportName)
		err := sendEmail(emailConfig)
		if err != nil {
			errOccurred = true
			fmt.Printf("sendEmail: %w\n", err)
		}
	}

	if errOccurred {
		return fmt.Errorf("Error occurred while sending reports")
	}
	return nil
}
