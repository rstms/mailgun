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
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/emersion/go-message/mail"
	"github.com/mailgun/mailgun-go/v5"
	"github.com/mailgun/mailgun-go/v5/events"
	"github.com/mailgun/mailgun-go/v5/mtypes"
	"github.com/spf13/viper"
)

type Client struct {
	domain string
	api    *mailgun.Client
	edb    *DB
	bdb    *DB
	mutex  sync.Mutex
}

func NewClient() *Client {
	viper.SetDefault("api_query_timeout", 30)
	client := Client{
		domain: viper.GetString("domain"),
		api:    mailgun.NewMailgun(viper.GetString("api_key")),
		edb:    NewDB(viper.GetString("data_root"), "mailgun.events"),
		bdb:    NewDB(viper.GetString("data_root"), "mailgun.bounced"),
	}
	return &client
}

func (c *Client) Domains() ([]string, error) {
	names := []string{}
	c.mutex.Lock()
	defer c.mutex.Unlock()
	domains := c.api.ListDomains(nil)
	var page []mtypes.Domain
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(viper.GetInt("api_query_timeout")))
	defer cancel()
	for domains.Next(ctx, &page) {
		for _, domain := range page {
			if domain.Type != "sandbox" && !domain.IsDisabled {
				names = append(names, domain.Name)
			}
		}
	}
	return names, nil
}

func (c *Client) ResetEvents() error {
	return c.edb.Reset()
}

func (c *Client) ResetBounced() error {
	return c.bdb.Reset()
}

func (c *Client) storeEvent(event events.Event) error {
	eid := event.GetID()
	if c.edb.Has(eid) {
		if viper.GetBool("verbose") {
			log.Printf("dup_event: %s\n", eid)
		}
	} else {
		err := c.edb.SetObject(eid, event)
		if err != nil {
			return err
		}
		if !viper.GetBool("quiet") {
			log.Printf("new_event %s %s %s\n", eid, event.GetName(), event.GetTimestamp().Format(time.RFC3339))
		}
	}
	return nil

}

func (c *Client) QueryBounceAddrs() (*[]mtypes.Bounce, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	iter := c.api.ListBounces(c.domain, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bounces := []mtypes.Bounce{}
	var page []mtypes.Bounce
	for iter.Next(ctx, &page) {
		for _, bounce := range page {
			if !viper.GetBool("quiet") {
				log.Printf("bounced_address: %s\n", bounce.Address)
			}
			bounces = append(bounces, bounce)
		}
	}
	if len(bounces) > 0 && !viper.GetBool("no_delete") {
		err := c.api.DeleteBounceList(ctx, c.domain)
		if err != nil {
			return nil, err
		}
		if !viper.GetBool("quiet") {
			log.Printf("deleted bounced address list (count=%d) for %s\n", len(bounces), c.domain)
		}
	}
	return &bounces, nil
}

func (c *Client) QueryEvents() (*[]events.Event, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	iter := c.api.ListEvents(c.domain, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	allEvents := []events.Event{}
	var page []events.Event
	for iter.Next(ctx, &page) {
		for _, event := range page {
			allEvents = append(allEvents, event)
			err := c.storeEvent(event)
			if err != nil {
				return nil, err
			}
		}
	}
	return &allEvents, nil
}

func (c *Client) MonitorEvents() error {
	bounceDisabled := viper.GetBool("no_bounce")
	options := mailgun.ListEventOptions{PollInterval: time.Second * time.Duration(viper.GetInt("poll_interval"))}
	iter := c.api.PollEvents(c.domain, &options)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var newEvents []events.Event
	for iter.Poll(ctx, &newEvents) {
		for _, event := range newEvents {
			err := c.storeEvent(event)
			if err != nil {
				return err
			}
		}
		if !bounceDisabled {
			err := c.SendBounces()
			if err != nil {
				return err
			}
		}
		_, err := c.QueryBounceAddrs()
		if err != nil {
			return err
		}
		err = c.PruneBounced()
		if err != nil {
			return err
		}
	}
	return fmt.Errorf("event poll failed")
}

func (c *Client) PruneEvents() error {
	err := c.ResetEvents()
	if err != nil {
		return err
	}
	_, err = c.QueryEvents()
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) PruneBounced() error {

	keys, err := c.bdb.Keys()
	if err != nil {
		return err
	}

	for _, key := range keys {
		if !c.edb.Has(key) {
			err := c.bdb.Clear(key)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Client) SendBounces() error {

	keys, err := c.edb.Keys()
	if err != nil {
		return err
	}
	for _, key := range keys {
		data, err := c.edb.Get(key)
		if err != nil {
			return err
		}
		event, err := events.ParseEvent(*data)
		if err != nil {
			return err
		}
		if event.GetName() == "failed" && !c.bdb.Has(key) {
			failed := event.(*events.Failed)
			err = c.sendBounce(failed)
			if err != nil {
				return err
			}
			if !viper.GetBool("quiet") {
				log.Printf("sent_bounce: %s <%s> %s\n", key, failed.Message.Headers.MessageID, failed.Recipient)
			}
			flag := true
			err = c.bdb.SetObject(key, &flag)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Client) addPart(mailWriter *mail.Writer, buf *bytes.Buffer) error {
	part, err := mailWriter.CreateInline()
	defer part.Close()
	if err != nil {
		return err
	}
	var header mail.InlineHeader
	header.Set("Content-Type", "text/plain")
	writer, err := part.CreatePart(header)
	defer writer.Close()
	if err != nil {
		return err
	}
	_, err = io.WriteString(writer, buf.String())
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) formatBounce(event *events.Failed, buf *bytes.Buffer) error {

	hostname, err := os.Hostname()
	if err != nil {
		return err
	}

	from := []*mail.Address{{Name: "Mailer Daemon", Address: fmt.Sprintf("MAILER-DAEMON@%s", hostname)}}
	to := []*mail.Address{{Address: event.Envelope.Sender}}

	var mailHeader mail.Header
	mailHeader.SetDate(time.Now())
	mailHeader.SetAddressList("From", from)
	mailHeader.SetAddressList("To", to)
	mailHeader.SetSubject("Delivery status notification: failed")

	mailWriter, err := mail.CreateWriter(buf, mailHeader)
	if err != nil {
		return err
	}
	defer mailWriter.Close()

	var pbuf bytes.Buffer

	pbuf.WriteString(`    Hi!

    This is the MAILER-DAEMON, please DO NOT REPLY to this email.

    An error has occurred while attempting to deliver a message
    for the following list of recipients:

`)
	pbuf.WriteString(fmt.Sprintf("%s: %v %s\n\n", event.Recipient, event.DeliveryStatus.Code, event.DeliveryStatus.Message))
	pbuf.WriteString("    Below is a copy of the original message:")
	err = c.addPart(mailWriter, &pbuf)
	if err != nil {
		return err
	}

	pbuf.Reset()
	pbuf.WriteString(fmt.Sprintf("Reporting-MTA: dns; %s; mailgun-relay\n\n", hostname))
	pbuf.WriteString(fmt.Sprintf("Final-Recipient: rfc822; %s\n", event.Recipient))
	pbuf.WriteString(fmt.Sprintf("Action: %s failure; %s\n", event.Severity, event.Reason))
	pbuf.WriteString(fmt.Sprintf("Status: %v", event.DeliveryStatus.Code))
	err = c.addPart(mailWriter, &pbuf)
	if err != nil {
		return err
	}

	pbuf.Reset()
	pbuf.WriteString(fmt.Sprintf("Subject: %s\n", event.Message.Headers.Subject))
	pbuf.WriteString(fmt.Sprintf("From: %s\n", event.Message.Headers.Subject))
	pbuf.WriteString(fmt.Sprintf("To: %s\n", event.Message.Headers.Subject))
	pbuf.WriteString(fmt.Sprintf("Message-ID <%s>", event.Message.Headers.MessageID))
	err = c.addPart(mailWriter, &pbuf)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) sendBounce(failed *events.Failed) error {

	var buf bytes.Buffer
	err := c.formatBounce(failed, &buf)
	if err != nil {
		return err
	}
	cmd := exec.Command("sendmail", "-t")
	cmd.Stdin = bytes.NewReader(buf.Bytes())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("sendmail failed: %s", string(output))
	}
	return nil
}
