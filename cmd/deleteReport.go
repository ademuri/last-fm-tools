/*
Copyright 2026 Google LLC

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

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// deleteReportCmd represents the deleteReport command
var deleteReportCmd = &cobra.Command{
	Use:   "delete-report",
	Short: "Deletes a report configured for the user",
	Long:  ``,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if viper.GetString("user") == "" {
			return fmt.Errorf("required flag(s) \"user\" not set")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		err := deleteReport(viper.GetString("database"), viper.GetString("user"), viper.GetString("name"), viper.GetString("dest"))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(deleteReportCmd)

	var reportName string
	deleteReportCmd.Flags().StringVar(&reportName, "name", "", "Name of the report to delete")
	deleteReportCmd.MarkFlagRequired("name")
	viper.BindPFlag("name", deleteReportCmd.Flags().Lookup("name"))

	var email string
	deleteReportCmd.Flags().StringVar(&email, "dest", "", "Destination email of the report to delete")
	deleteReportCmd.MarkFlagRequired("dest")
	viper.BindPFlag("dest", deleteReportCmd.Flags().Lookup("dest"))
}

func deleteReport(dbPath string, user string, name string, email string) error {
	db, err := openDb(dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	res, err := db.Exec("DELETE FROM Report WHERE user = ? AND name = ? AND email = ?", user, name, email)
	if err != nil {
		return fmt.Errorf("delete report: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no report found with name %q and email %q for user %q", name, email, user)
	}

	fmt.Printf("Deleted report %q (%s) for user %q\n", name, email, user)
	return nil
}
