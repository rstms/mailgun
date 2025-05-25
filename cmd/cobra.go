package cmd

import (
	"github.com/spf13/viper"
)

var ViperPrefix = ""

func OptionSwitch(name, flag, description string) {

	if flag == "" {
		rootCmd.PersistentFlags().Bool(name, false, description)
	} else {
		rootCmd.PersistentFlags().BoolP(name, flag, false, description)
	}

	viper.BindPFlag(ViperPrefix+name, rootCmd.PersistentFlags().Lookup(name))
}

func OptionString(name, flag, defaultValue, description string) {

	if flag == "" {
		rootCmd.PersistentFlags().String(name, defaultValue, description)
	} else {
		rootCmd.PersistentFlags().StringP(name, flag, defaultValue, description)
	}

	viper.BindPFlag(ViperPrefix+name, rootCmd.PersistentFlags().Lookup(name))
}
