package cmd

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func initTestConfig() {
	cfgFile = "testdata/config.yaml"
	initConfig()
}

func TestEvents(t *testing.T) {
	initTestConfig()
	api := InitAPI()
	err := QueryEvents(api)
	require.Nil(t, err)
}

func TestBounces(t *testing.T) {
	initTestConfig()
	err := GenerateBounces()
	require.Nil(t, err)
}
