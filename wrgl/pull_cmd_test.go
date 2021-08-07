package main

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apitest "github.com/wrgl/core/pkg/api/test"
	"github.com/wrgl/core/pkg/factory"
	objmock "github.com/wrgl/core/pkg/objects/mock"
	"github.com/wrgl/core/pkg/ref"
	refmock "github.com/wrgl/core/pkg/ref/mock"
)

func TestPullCmd(t *testing.T) {
	dbs := objmock.NewStore()
	rss := refmock.NewStore()
	sum1, c1 := factory.CommitRandom(t, dbs, nil)
	sum2, c2 := factory.CommitRandom(t, dbs, [][]byte{sum1})
	require.NoError(t, ref.CommitHead(rss, "main", sum2, c2))
	sum4, c4 := factory.CommitRandom(t, dbs, nil)
	sum5, c5 := factory.CommitRandom(t, dbs, [][]byte{sum4})
	require.NoError(t, ref.CommitHead(rss, "beta", sum5, c5))
	ts := uploadPackServer(rss, dbs)
	defer ts.Close()

	rd, cleanUp := createRepoDir(t)
	defer cleanUp()
	db, err := rd.OpenObjectsStore()
	require.NoError(t, err)
	rs := rd.OpenRefStore()
	apitest.CopyCommitsToNewStore(t, dbs, db, [][]byte{sum1, sum4})
	require.NoError(t, ref.CommitHead(rs, "main", sum1, c1))
	require.NoError(t, ref.CommitHead(rs, "beta", sum4, c4))
	require.NoError(t, db.Close())

	cmd := newRootCmd()
	cmd.SetArgs([]string{"remote", "add", "origin", ts.URL, "-t", "beta", "-t", "main"})
	require.NoError(t, cmd.Execute())

	// pull set upstream
	cmd = newRootCmd()
	cmd.SetArgs([]string{"pull", "main", "origin", "refs/heads/main:refs/remotes/origin/main", "--set-upstream"})
	cmd.SetOut(io.Discard)
	require.NoError(t, cmd.Execute())

	db, err = rd.OpenObjectsStore()
	require.NoError(t, err)
	sum, err := ref.GetHead(rs, "main")
	require.NoError(t, err)
	assert.Equal(t, sum2, sum)
	require.NoError(t, db.Close())

	sum3, c3 := factory.CommitRandom(t, dbs, [][]byte{sum2})
	require.NoError(t, ref.CommitHead(rss, "main", sum3, c3))

	// pull with upstream already set
	cmd = newRootCmd()
	cmd.SetArgs([]string{"pull", "main"})
	cmd.SetOut(io.Discard)
	require.NoError(t, cmd.Execute())

	db, err = rd.OpenObjectsStore()
	require.NoError(t, err)
	sum, err = ref.GetHead(rs, "main")
	require.NoError(t, err)
	assert.Equal(t, sum3, sum)
	require.NoError(t, db.Close())

	// pull merge first fetch refspec
	cmd = newRootCmd()
	cmd.SetArgs([]string{"pull", "beta"})
	cmd.SetOut(io.Discard)
	require.NoError(t, cmd.Execute())

	db, err = rd.OpenObjectsStore()
	require.NoError(t, err)
	sum, err = ref.GetHead(rs, "beta")
	require.NoError(t, err)
	assert.Equal(t, sum5, sum)
	require.NoError(t, db.Close())

	cmd = newRootCmd()
	cmd.SetArgs([]string{"pull", "main"})
	assertCmdOutput(t, cmd, "Already up to date.\n")
}
