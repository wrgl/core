// SPDX-License-Identifier: Apache-2.0
// Copyright © 2022 Wrangle Ltd

package wrgl

import (
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	confhelpers "github.com/wrgl/wrgl/pkg/conf/helpers"
	"github.com/wrgl/wrgl/pkg/testutils"
)

func TestConfigSetCmd(t *testing.T) {
	defer confhelpers.MockGlobalConf(t, true)()
	defer confhelpers.MockSystemConf(t)()
	wrglDir, err := testutils.TempDir("", ".wrgl*")
	require.NoError(t, err)
	defer os.RemoveAll(wrglDir)
	viper.Set("wrgl_dir", wrglDir)

	cmd := rootCmd()
	cmd.SetArgs([]string{"config", "set", "user.name", "John Doe"})
	require.NoError(t, cmd.Execute())

	cmd = rootCmd()
	cmd.SetArgs([]string{"config", "set", "user.name", "John Smith", "--global"})
	require.NoError(t, cmd.Execute())

	cmd = rootCmd()
	cmd.SetArgs([]string{"config", "set", "user.name", "Jane Lane", "--system"})
	require.NoError(t, cmd.Execute())

	cmd = rootCmd()
	cmd.SetArgs([]string{"config", "get", "user.name", "--local"})
	assertCmdOutput(t, cmd, "John Doe\n")

	cmd = rootCmd()
	cmd.SetArgs([]string{"config", "get", "user.name", "--system"})
	assertCmdOutput(t, cmd, "Jane Lane\n")

	cmd = rootCmd()
	cmd.SetArgs([]string{"config", "get", "user.name", "--global"})
	assertCmdOutput(t, cmd, "John Smith\n")

	cmd = rootCmd()
	cmd.SetArgs([]string{"config", "get", "user.name"})
	assertCmdOutput(t, cmd, "John Doe\n")

	cmd = rootCmd()
	cmd.SetArgs([]string{"config", "set", "merge.fastForward", "only"})
	require.NoError(t, cmd.Execute())

	cmd = rootCmd()
	cmd.SetArgs([]string{"config", "get", "merge.fastForward"})
	assertCmdOutput(t, cmd, "only\n")
}

func TestConfigSetCmdBool(t *testing.T) {
	defer confhelpers.MockGlobalConf(t, true)()
	defer confhelpers.MockSystemConf(t)()
	wrglDir, err := testutils.TempDir("", ".wrgl*")
	require.NoError(t, err)
	defer os.RemoveAll(wrglDir)
	viper.Set("wrgl_dir", wrglDir)

	cmd := rootCmd()
	cmd.SetArgs([]string{"config", "set", "receive.denyDeletes", "true"})
	require.NoError(t, cmd.Execute())

	cmd = rootCmd()
	cmd.SetArgs([]string{"config", "set", "receive.denyNonFastForwards", "false"})
	require.NoError(t, cmd.Execute())

	cmd = rootCmd()
	cmd.SetArgs([]string{"config", "get", "receive.denyDeletes"})
	assertCmdOutput(t, cmd, "true\n")

	cmd = rootCmd()
	cmd.SetArgs([]string{"config", "get", "receive.denyNonFastForwards"})
	assertCmdOutput(t, cmd, "false\n")
}
