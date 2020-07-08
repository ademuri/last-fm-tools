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
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type SendReportsConfig struct {
	DbPath string
	From   string
	DryRun bool
}

var sendReportsCmd = &cobra.Command{
	Use:   "send-reports",
	Short: "Update the database and send email reports.",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		config := SendReportsConfig{
			DbPath: viper.GetString("database"),
			From:   viper.GetString("from"),
			DryRun: viper.GetBool("dry_run"),
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

	var from string
	sendReportsCmd.Flags().StringVar(&from, "from", "", "From email address")
	sendReportsCmd.MarkFlagRequired("from")
	viper.BindPFlag("from", sendReportsCmd.Flags().Lookup("from"))

	var dryRun bool
	sendReportsCmd.Flags().BoolVarP(&dryRun, "dry_run", "n", false, "When true, just print instead of emailing")
	viper.BindPFlag("dry_run", sendReportsCmd.Flags().Lookup("dry_run"))
}

func sendReports(config SendReportsConfig) error {
	db, err := createDatabase(config.DbPath)
	if err != nil {
		return err
	}
	reports, err := db.Query("SELECT name, user, email, sent, run_day, types FROM Report")
	if err != nil {
		return fmt.Errorf("Querying reports: %w", err)
	}

	emailConfigs := make([]SendEmailConfig, 0)
	for reports.Next() {
		var (
			sentOrNull sql.NullTime
			runDay     int
			types      string
		)

		now := time.Now()
		emailConfig := SendEmailConfig{
			DbPath: config.DbPath,
			From:   config.From,
			DryRun: config.DryRun,
		}
		err = reports.Scan(&emailConfig.ReportName, &emailConfig.User, &emailConfig.To, &sentOrNull, &runDay, &types)
		if err != nil {
			return fmt.Errorf("Getting report params: %w", err)
		}
		var sent time.Time
		if sentOrNull.Valid {
			sent = sentOrNull.Time
		}

		emailConfig.Types = strings.Split(types, ",")
		toSendThisMonth := time.Date(now.Year(), now.Month(), runDay, 0, 0, 0, 0, now.Location())
		toSendLastMonth := time.Date(now.Year(), now.Month()-1, runDay, 0, 0, 0, 0, now.Location())
		if sent.After(toSendThisMonth) {
			fmt.Printf("Report (%q, %q) was already sent this month on %s, not sending.\n", emailConfig.User, emailConfig.ReportName, sent.Format("2006-01-02"))
			continue
		}
		if now.Before(toSendThisMonth) && sent.After(toSendLastMonth) {
			fmt.Printf("Report (%q, %q) was already sent for last month on %s, not sending.\n", emailConfig.User, emailConfig.ReportName, sent.Format("2006-01-02"))
			continue
		}

		emailConfigs = append(emailConfigs, emailConfig)
	}
	reports.Close()

	errOccurred := false
	for _, emailConfig := range emailConfigs {
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
