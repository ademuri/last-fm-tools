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
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/ademuri/lastfm-go/lastfm"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var authenticateCmd = &cobra.Command{
	Use:   "authenticate <email> --user=foo",
	Short: "Gets a session key for the given user.",
	Long:  `This is needed if the user has marked their data as private.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		err := getSessionKey(viper.GetString("database"), viper.GetString("from"), args)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(authenticateCmd)

	var from string
	authenticateCmd.Flags().StringVar(&from, "from", "", "From email address")
	viper.BindPFlag("from", authenticateCmd.Flags().Lookup("from"))
}

func getSessionKey(dbPath string, fromAddress string, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("Expected exactly one email argument")
	}

	user := strings.ToLower(viper.GetString("user"))
	db, err := createDatabase(dbPath)

	userQuery, err := db.Query("SELECT session_key FROM User WHERE name = ? AND session_key <> ''", user)
	if err != nil {
		return fmt.Errorf("Getting existing session_key: %w", err)
	}
	if userQuery.Next() {
		return fmt.Errorf("User %s already has session key", user)
	}
	userQuery.Close()

	lastfmClient := lastfm.New(lastFmApiKey, lastFmSecret)
	lastfmClient.SetUserAgent("last-fm-tools/1.0")

	authToken, err := lastfmClient.GetToken()
	if err != nil {
		return fmt.Errorf("Getting token: %w")
	}

	authUrl := lastfmClient.GetAuthTokenUrl(authToken)
	if err != nil {
		return fmt.Errorf("Getting token URL: %w")
	}

	toAddress := args[0]
	from := mail.NewEmail("last-fm-tools", fromAddress)
	subject := "Authenticate last-fm-tools"
	to := mail.NewEmail(toAddress, toAddress)
	bodyText := "Click here to authenticate: " + authUrl
	message := mail.NewSingleEmail(from, subject, to, bodyText, bodyText)
	client := sendgrid.NewSendClient(viper.GetString("sendgrid_api_key"))
	_, err = client.Send(message)
	if err != nil {
		return fmt.Errorf("sendEmail: %w", err)
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Sent authentication email, press the anykey to continue")
	reader.ReadString('\n')

	lastfmClient.LoginWithToken(authToken)
	if err != nil {
		return fmt.Errorf("Logging in: %w", err)
	}
	sessionKey := lastfmClient.GetSessionKey()

	_, err = db.Exec("UPDATE User SET session_key = ? WHERE name = ?", sessionKey, user)
	if err != nil {
		return fmt.Errorf("Updating db with session key: %w", err)
	}

	fmt.Println("Successfully authenticated %q", user)
	return nil
}
