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
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"strings"
)

const Version = "0.1.1"

var cfgFile string
var logFile *os.File

var rootCmd = &cobra.Command{
	Use:   "mailgun",
	Short: "mailgun toolkit",
	Long: `
Functions making use of the mailgun API
`,
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if logFile != nil {
			err := logFile.Close()
			cobra.CheckErr(err)
			logFile = nil
		}
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
func init() {
	cobra.OnInitialize(initConfig)
	OptionSwitch("verbose", "v", "enable diagnostic output")
	OptionSwitch("foreground", "", "do not daemonize monitor")
	OptionSwitch("quiet", "q", "suppress non-error output")
	OptionSwitch("json", "j", "output JSON objects")
	OptionSwitch("no-bounce", "", "disable automatic bounce generation")
	OptionSwitch("no-delete", "", "disable deletion of bounced addresses")
	hostname, err := os.Hostname()
	cobra.CheckErr(err)
	_, domain, _ := strings.Cut(hostname, ".")
	OptionString("domain", "d", domain, "mailgun domain")
	cacheDir, err := os.UserCacheDir()
	cobra.CheckErr(err)
	OptionString("data-root", "", filepath.Join(cacheDir, "mailgun"), "database root directory")
	OptionString("poll-interval", "", "5", "event poll interval seconds")
	OptionString("logfile", "l", "stderr", "log file")
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file")
}
func initConfig() {
	configDir, err := os.UserConfigDir()
	cobra.CheckErr(err)
	systemConfigFile := "/etc/mailgun/config.yaml"
	userConfigFile := filepath.Join(configDir, "mailgun", "config.yaml")
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else if IsFile(systemConfigFile) {
		viper.SetConfigFile(systemConfigFile)
	} else if IsFile(userConfigFile) {
		viper.SetConfigFile(userConfigFile)
	} else {
		homeDir, err := os.UserHomeDir()
		cobra.CheckErr(err)
		viper.AddConfigPath(homeDir)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".mailgun")
	}
	viper.SetEnvPrefix("mailgun")
	if err := viper.ReadInConfig(); err == nil {
		if viper.GetBool("verbose") {
			fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		}
	}
	InitLog()
}
