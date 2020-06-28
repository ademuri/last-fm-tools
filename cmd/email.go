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

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var emailCmd = &cobra.Command{
	Use:   "email <address>",
	Short: "Sends an email report",
	Long:  `Emails history to the specified user.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		err := sendEmail(viper.GetString("database"), viper.GetString("from"), args)
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
}

func sendEmail(dbPath string, fromAddress string, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("Expected exactly one recipient email address")
	}

	from := mail.NewEmail("last-fm-tools", fromAddress)
	subject := "last.fm report"
	to := mail.NewEmail(args[0], args[0])
	body := "My first email"
	message := mail.NewSingleEmail(from, subject, to, body, body)
	client := sendgrid.NewSendClient(viper.GetString("sendgrid_api_key"))
	_, err := client.Send(message)
	if err != nil {
		return fmt.Errorf("email: %w", err)
	}

	return nil
}
