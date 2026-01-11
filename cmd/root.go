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

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var cfgFile string
var lastFmApiKey string
var lastFmSecret string
var lastFmUser string
var databasePath string
var smtpUsername string
var smtpPassword string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "last-fm-tools",
	Short: "Performs analysis on last.fm listening data",
	Long:  `Someday, this will do things.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(
		&cfgFile, "config", "", "config file (default is $HOME/.last-fm-tools.yaml)")

	rootCmd.PersistentFlags().StringVarP(
		&lastFmApiKey, "api_key", "", "", "last.fm API key")
	rootCmd.MarkPersistentFlagRequired("api_key")
	viper.BindPFlag("api_key", rootCmd.PersistentFlags().Lookup("api_key"))

	rootCmd.PersistentFlags().StringVarP(
		&lastFmSecret, "secret", "", "", "last.fm secret")
	rootCmd.MarkPersistentFlagRequired("secret")
	viper.BindPFlag("secret", rootCmd.PersistentFlags().Lookup("secret"))

	rootCmd.PersistentFlags().StringVarP(
		&lastFmUser, "user", "u", "", "last.fm username to act on")
	rootCmd.MarkPersistentFlagRequired("user")
	viper.BindPFlag("user", rootCmd.PersistentFlags().Lookup("user"))

	rootCmd.PersistentFlags().StringVarP(
		&databasePath, "database", "d", "./lastfm.db", "Path to the SQLite database")
	viper.BindPFlag("database", rootCmd.PersistentFlags().Lookup("database"))

	rootCmd.PersistentFlags().StringVar(&smtpUsername, "smtp_username", "", "SMTP username")
	viper.BindPFlag("smtp_username", rootCmd.PersistentFlags().Lookup("smtp_username"))

	rootCmd.PersistentFlags().StringVar(&smtpPassword, "smtp_password", "", "SMTP password")
	viper.BindPFlag("smtp_password", rootCmd.PersistentFlags().Lookup("smtp_password"))

	var from string
	rootCmd.PersistentFlags().StringVar(&from, "from", "", "From email address")
	rootCmd.MarkPersistentFlagRequired("from")
	viper.BindPFlag("from", rootCmd.PersistentFlags().Lookup("from"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".last-fm-tools" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".last-fm-tools")
	}

	// viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}

	// See https://github.com/spf13/viper/pull/852
	rootCmd.Flags().VisitAll(func(f *pflag.Flag) {
		if viper.IsSet(f.Name) && viper.GetString(f.Name) != "" {
			rootCmd.Flags().Set(f.Name, viper.GetString(f.Name))
		}
	})
}

func openDb(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("openDb: %w", err)
	}
	return db, nil
}

func dbExists(db *sql.DB) (bool, error) {
	exists, err := db.Query("SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'User'")
	if err != nil {
		return false, fmt.Errorf("createTables: %w", err)
	}
	defer exists.Close()

	return exists.Next(), nil
}
