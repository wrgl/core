// SPDX-License-Identifier: Apache-2.0
// Copyright © 2021 Wrangle Ltd

package remote

import (
	"github.com/spf13/cobra"
	"github.com/wrgl/core/pkg/versioning"
	"github.com/wrgl/core/wrgl/utils"
)

func removeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove NAME",
		Aliases: []string{"rm"},
		Short:   "Remove the remote named NAME.",
		Long:    "All remote-tracking branches and configuration settings for the remote are removed.",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			wrglDir := utils.MustWRGLDir(cmd)
			c, err := versioning.OpenConfig(false, false, wrglDir, "")
			if err != nil {
				return err
			}
			rd := versioning.NewRepoDir(wrglDir, false, false)
			db, err := rd.OpenKVStore()
			if err != nil {
				return err
			}
			defer db.Close()
			fs := rd.OpenFileStore()
			err = versioning.DeleteAllRemoteRefs(db, fs, args[0])
			if err != nil {
				return err
			}
			delete(c.Remote, args[0])
			return c.Save()
		},
	}
	return cmd
}
