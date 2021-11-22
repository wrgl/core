// SPDX-License-Identifier: Apache-2.0
// Copyright © 2021 Wrangle Ltd

package config

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wrgl/wrgl/cmd/wrgl/utils"
	"github.com/wrgl/wrgl/pkg/dotno"
)

func replaceAllCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "replace-all NAME VALUE [VALUE_PATTERN]",
		Short: "Replace all values with a single value.",
		Long:  "Replace all values with a single value. If VALUE_PATTERN is given, only replace the values matching it.",
		Example: utils.CombineExamples([]utils.Example{
			{
				Comment: "replace all values under remote.origin.push with refs/heads/main",
				Line:    "wrgl config replace-all remote.origin.push refs/heads/main",
			},
			{
				Comment: "replace all branches under remote.origin.push with refs/heads/main",
				Line:    "wrgl config replace-all remote.origin.push refs/heads/main ^refs/heads/",
			},
		}),
		Args: cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := utils.MustWRGLDir(cmd)
			s := writeableConfigStore(cmd, dir)
			c, err := s.Open()
			if err != nil {
				return err
			}
			v, err := dotno.GetFieldValue(c, args[0], true)
			if err != nil {
				return err
			}
			if len(args) > 2 {
				idxMap, _, err := dotno.FilterWithValuePattern(cmd, v, args[2])
				if err != nil {
					return err
				}
				if sl, ok := dotno.ToTextSlice(v.Interface()); ok {
					result := []string{}
					n := sl.Len()
					for i := 0; i < n; i++ {
						if _, ok := idxMap[i]; !ok {
							s, err := sl.Get(i)
							if err != nil {
								return err
							}
							result = append(result, s)
						}
					}
					result = append(result, args[1])
					sl, err = dotno.TextSliceFromStrSlice(v.Type(), result)
					if err != nil {
						return err
					}
					v.Set(sl.Value)
				} else {
					panic(fmt.Sprintf("type %v does not implement encoding.TextUnmarshaler", v.Type().Elem()))
				}
			} else {
				err = dotno.SetValue(v, args[1], true)
				if err != nil {
					return err
				}
			}
			return s.Save(c)
		},
	}
	return cmd
}
