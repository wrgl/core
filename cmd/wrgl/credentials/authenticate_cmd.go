// SPDX-License-Identifier: Apache-2.0
// Copyright © 2021 Wrangle Ltd

package credentials

import (
	"bufio"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wrgl/core/cmd/wrgl/utils"
	apiclient "github.com/wrgl/core/pkg/api/client"
	conffs "github.com/wrgl/core/pkg/conf/fs"
	"github.com/wrgl/core/pkg/credentials"
)

func authenticateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "authenticate { REMOTE_URI | REMOTE_NAME }",
		Short: "Authenticate for one or more remotes with email/password.",
		Long:  "Authenticate for one or more remotes with email/password and save credentials for future use. If REMOTE_NAME is given, then login and save credentials for that remote. If REMOTE_URI is given, login at REMOTE_URI/authenticate/ and save credentials for all remotes that have REMOTE_URI as prefix.",
		Example: utils.CombineExamples([]utils.Example{
			{
				Comment: "authenticate for origin",
				Line:    "wrgl credentials authenticate origin",
			},
			{
				Comment: "authenticate for all repositories on wrgl hub",
				Line:    "wrgl credentials authenticate https://hub.wrgl.co/api",
			},
		}),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := utils.MustWRGLDir(cmd)
			cfs := conffs.NewStore(dir, conffs.LocalSource, "")
			c, err := cfs.Open()
			if err != nil {
				return err
			}
			cs, err := credentials.NewStore()
			if err != nil {
				return err
			}
			if v, ok := c.Remote[args[0]]; ok {
				_, _, err = getCredentials(cmd, cs, v.URL)
			} else {
				_, _, err = getCredentials(cmd, cs, args[0])
			}
			return err
		},
	}
	return cmd
}

func getCredentials(cmd *cobra.Command, cs *credentials.Store, uriS string) (uri *url.URL, token string, err error) {
	u, err := url.Parse(uriS)
	if err != nil {
		return
	}
	cmd.Printf("Enter your email and password for %s.\n", uriS)
	reader := bufio.NewReader(cmd.InOrStdin())
	cmd.Print("Email: ")
	email, err := reader.ReadString('\n')
	if err != nil {
		return nil, "", err
	}
	email = strings.TrimSpace(email)
	password, err := utils.PromptForPassword(cmd)
	if err != nil {
		return nil, "", err
	}
	client, err := apiclient.NewClient(uriS)
	if err != nil {
		return nil, "", err
	}
	token, err = client.Authenticate(email, password)
	if err != nil {
		return nil, "", err
	}
	cs.Set(*u, token)
	if err := cs.Flush(); err != nil {
		return nil, "", err
	}
	cmd.Printf("Saved credentials to %s\n", cs.Path())
	uri = u
	return
}
