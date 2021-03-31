package main

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wrgl/core/pkg/factory"
	"github.com/wrgl/core/pkg/versioning"
)

func TestResetCmd(t *testing.T) {
	rd, cleanUp := createRepoDir(t)
	defer cleanUp()
	cmd := newRootCmd()

	db, err := rd.OpenKVStore()
	require.NoError(t, err)
	sum, _ := factory.CommitSmall(t, db, "alpha", nil, nil, nil)
	factory.CommitSmall(t, db, "alpha", nil, nil, nil)
	require.NoError(t, db.Close())

	setCmdArgs(cmd, rd, "reset", "alpha", sum)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	require.NoError(t, cmd.Execute())

	db, err = rd.OpenKVStore()
	require.NoError(t, err)
	defer db.Close()
	b, err := versioning.GetBranch(db, "alpha")
	require.NoError(t, err)
	assert.Equal(t, sum, b.CommitHash)
}
