package cmd

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

func initTestConfig() {
	cfgFile = "testdata/config.yaml"
	initConfig()
}

func TestEvents(t *testing.T) {
	initTestConfig()
	api := NewClient()
	events, err := api.QueryEvents()
	require.Nil(t, err)
	fmt.Printf("%v\n", events)
}
