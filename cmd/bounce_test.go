package cmd

import (
	"fmt"
	"path/filepath"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestScanCommand(t *testing.T) {
	cfgFile = "testdata/config.yaml"
	initConfig()

	err := InitIdentity()
	require.Nil(t, err)
	filterctl := NewFilterctlClient()
	sender := "sender@example.org"
	address := "address@example.org"
	viper.Set("message_id", EncodedMessageID("test scan message id"))
	var response api.BooksResponse
	path := fmt.Sprintf("/filterctl/scan/%s/%s/", sender, address)
	text, err := filterctl.Get(path, &response)
	require.Nil(t, err)
	fmt.Printf("text=%v\n", text)
	fmt.Printf("response=%v\n", response)
}
