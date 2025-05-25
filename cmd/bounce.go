/*
Copyright © 2025 Matt Krueger <mkrueger@rstms.net>
All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

 1. Redistributions of source code must retain the above copyright notice,
    this list of conditions and the following disclaimer.

 2. Redistributions in binary form must reproduce the above copyright notice,
    this list of conditions and the following disclaimer in the documentation
    and/or other materials provided with the distribution.

 3. Neither the name of the copyright holder nor the names of its contributors
    may be used to endorse or promote products derived from this software
    without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE
LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
POSSIBILITY OF SUCH DAMAGE.
*/
package cmd

import (
	"fmt"

	"github.com/mailgun/mailgun-go/v5/events"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// bounceCmd represents the bounce command
var bounceCmd = &cobra.Command{
	Use:   "bounce",
	Short: "generate and send bounce emails",
	Long: `
Scan all event files in the mailgun.events store.  For each 'failure' event
that is not present in the mailgun.bounces store, generate and send a bounce
message.  After sending the bounce, write the key into the mailgun.bounced
store.
`,
	Run: func(cmd *cobra.Command, args []string) {
		edb, err := NewDB("mailgun.events", viper.GetString("data_dir"))
		bdb, err := NewDB("mailgun.bounced", viper.GetString("data_dir"))
		keys, err := edb.Keys()
		cobra.CheckErr(err)
		for _, key := range keys {
			data, err := edb.Get(key)
			cobra.CheckErr(err)
			event, err := events.ParseEvent(*data)
			cobra.CheckErr(err)
			if event.GetName() == "failed" && !bdb.Has(key) {
				err := sendBounce(event.(*events.Failed))
				cobra.CheckErr(err)
				//flag := true
				//err = bdb.SetObject(key, &flag)
				//cobra.CheckErr(err)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(bounceCmd)
}

const BounceOpening = `Hi!

This is the MAILER-DAEMON, please DO NOT REPLY to this email.

An error has occurred while attempting to deliver a message
for the following list of recipients:

`

func sendBounce(event *events.Failed) error {
	headers := event.Message.Headers
	fmt.Printf("---bounce---\nreason=%s\nheaders=%+v\n", event.Reason, headers)
	fmt.Println(formatJSON(event))
	return nil
}
