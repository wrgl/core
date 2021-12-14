// SPDX-License-Identifier: Apache-2.0
// Copyright © 2021 Wrangle Ltd

package wrgl

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/wrgl/wrgl/cmd/wrgl/utils"
	"github.com/wrgl/wrgl/pkg/local"
)

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a repository in the working directory.",
		Example: utils.CombineExamples([]utils.Example{
			{
				Comment: "initialize repository at <working directory>/.wrgl",
				Line:    "wrgl init",
			},
			{
				Comment: "initialize at directory \"my-repo\"",
				Line:    "wrgl init --wrgl-dir my-repo",
			},
		}),
		Args: cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			dir, err := cmd.Flags().GetString("wrgl-dir")
			if err != nil {
				return err
			}
			if dir == "" {
				dir = filepath.Join(wd, ".wrgl")
			}
			badgerLog, err := cmd.Flags().GetString("badger-log")
			if err != nil {
				return err
			}
			rd := local.NewRepoDir(dir, badgerLog)
			defer rd.Close()
			err = rd.Init()
			if err != nil {
				return err
			}
			cmd.Printf("Repository initialized at %s\n", dir)
			return nil
		},
	}
	return cmd
}
