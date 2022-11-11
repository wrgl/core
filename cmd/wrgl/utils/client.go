package utils

import (
	"fmt"

	"github.com/spf13/cobra"
	apiclient "github.com/wrgl/wrgl/pkg/api/client"
)

func GetClient(cmd *cobra.Command, url string, opts ...apiclient.ClientOption) (*apiclient.Client, error) {
	logger := GetLogger()
	client, err := apiclient.NewClient(url, logger, opts...)
	if err != nil {
		return nil, fmt.Errorf("error creating new client: %w", err)
	}
	return client, nil
}
