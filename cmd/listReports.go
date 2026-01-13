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
	"database/sql"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// listReportsCmd represents the listReports command
var listReportsCmd = &cobra.Command{
	Use:   "list-reports",
	Short: "Lists all reports configured for the user",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		err := listReports(viper.GetString("database"), viper.GetString("user"))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(listReportsCmd)
}

func listReports(dbPath string, user string) error {
	db, err := openDb(dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	var rows *sql.Rows
	if user != "" {
		rows, err = db.Query("SELECT user, name, email, run_day, types, params FROM Report WHERE user = ?", user)
	} else {
		rows, err = db.Query("SELECT user, name, email, run_day, types, params FROM Report")
	}
	if err != nil {
		return fmt.Errorf("query reports: %w", err)
	}
	defer rows.Close()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "USER\tNAME\tEMAIL\tRUN DAY\tTYPES\tPARAMS")

	for rows.Next() {
		var u string
		var name string
		var email string
		var runDay int
		var types string
		var params sql.NullString
		if err := rows.Scan(&u, &name, &email, &runDay, &types, &params); err != nil {
			return fmt.Errorf("scan report: %w", err)
		}
		pStr := ""
		if params.Valid {
			pStr = params.String
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\n", u, name, email, runDay, types, pStr)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate reports: %w", err)
	}

	w.Flush()
	return nil
}
