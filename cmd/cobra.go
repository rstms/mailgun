package cmd

import (
	"github.com/spf13/viper"
)

var ViperPrefix = ""

func GlobalSwitch(name, flag, description string) {

	if flag == "" {
		rootCmd.PersistentFlags().Bool(name, false, description)
	} else {
		rootCmd.PersistentFlags().BoolP(name, flag, false, description)
	}

	viper.BindPFlag(ViperPrefix+name, rootCmd.PersistentFlags().Lookup(name))
}
