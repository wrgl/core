// SPDX-License-Identifier: Apache-2.0
// Copyright © 2021 Wrangle Ltd

package remote

import (
	"github.com/spf13/cobra"
	"github.com/wrgl/core/cmd/wrgl/utils"
	conffs "github.com/wrgl/core/pkg/conf/fs"
)

func getURLCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-url NAME",
		Short: "Print the URL for a remote.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			wrglDir := utils.MustWRGLDir(cmd)
			s := conffs.NewStore(wrglDir, conffs.LocalSource, "")
			c, err := s.Open()
			if err != nil {
				return err
			}
			rem := utils.MustGetRemote(cmd, c, name)
			cmd.Println(rem.URL)
			return nil
		},
	}
	return cmd
}
