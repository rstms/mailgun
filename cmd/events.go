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
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mailgun/mailgun-go/v5"
	"github.com/mailgun/mailgun-go/v5/events"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// eventsCmd represents the events command
var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "query mailgun events",
	Long:  `Output mailgun events for selected domain.`,
	Run: func(cmd *cobra.Command, args []string) {
		API := mailgun.NewMailgun(viper.GetString("api_key"))
		domain := viper.GetString("domain")
		iter := API.ListEvents(domain, nil)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		db, err := NewDB("mailgun.events", viper.GetString("data_dir"))
		cobra.CheckErr(err)
		var allEvents []events.Event
		var page []events.Event
		for iter.Next(ctx, &page) {
			for _, event := range page {
				if viper.GetBool("json") {
					allEvents = append(allEvents, event)
				} else {
					fmt.Printf("%s\t%s\t%s\n", event.GetTimestamp().Format(time.RFC3339), event.GetID(), event.GetName())
				}
				err := db.SetObject(event.GetID(), &event)
				cobra.CheckErr(err)
			}
		}
		if viper.GetBool("json") {
			fmt.Println(formatJSON(&allEvents))
		}
	},
}

func init() {
	rootCmd.AddCommand(eventsCmd)
}

func formatJSON(v any) string {
	data, err := json.MarshalIndent(v, "", "  ")
	cobra.CheckErr(err)
	return string(data)
}
