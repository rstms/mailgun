package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
	"log"
	"os"
	"strings"
)

func IsDir(path string) bool {
	stat, err := os.Stat(path)
	if err != nil {
		return false
	}
	return stat.IsDir()
}

func IsFile(pathname string) bool {
	_, err := os.Stat(pathname)
	return !os.IsNotExist(err)
}

var ViperPrefix = ""

func ViperKey(name string) string {
	return ViperPrefix + strings.ReplaceAll(name, "-", "_")
}

func OptionSwitch(name, flag, description string) {

	if flag == "" {
		rootCmd.PersistentFlags().Bool(name, false, description)
	} else {
		rootCmd.PersistentFlags().BoolP(name, flag, false, description)
	}

	viper.BindPFlag(ViperKey(name), rootCmd.PersistentFlags().Lookup(name))
}

func OptionString(name, flag, defaultValue, description string) {

	if flag == "" {
		rootCmd.PersistentFlags().String(name, defaultValue, description)
	} else {
		rootCmd.PersistentFlags().StringP(name, flag, defaultValue, description)
	}

	viper.BindPFlag(ViperKey(name), rootCmd.PersistentFlags().Lookup(name))
}

func InitLog() {
	filename := viper.GetString("logfile")
	logFile = nil
	if filename == "stdout" || filename == "-" {
		log.SetOutput(os.Stdout)
	} else if filename == "stderr" {
		log.SetOutput(os.Stderr)
	} else {
		fp, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0660)
		if err != nil {
			log.Fatalf("failed opening log file: %v", err)
		}
		logFile = fp
		log.SetOutput(logFile)
	}
	log.SetPrefix(fmt.Sprintf("[%d] ", os.Getpid()))
	log.SetFlags(log.Ldate | log.Ltime | log.Lmsgprefix)
}

func FormatJSON(v any) string {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Fatalf("failed formatting JSON: %v", err)
	}
	return string(data)
}
