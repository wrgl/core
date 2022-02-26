// SPDX-License-Identifier: Apache-2.0
// Copyright © 2021 Wrangle Ltd

package api_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiclient "github.com/wrgl/wrgl/pkg/api/client"
	apitest "github.com/wrgl/wrgl/pkg/api/test"
	apiutils "github.com/wrgl/wrgl/pkg/api/utils"
	"github.com/wrgl/wrgl/pkg/conf"
	objmock "github.com/wrgl/wrgl/pkg/objects/mock"
	"github.com/wrgl/wrgl/pkg/ref"
	refmock "github.com/wrgl/wrgl/pkg/ref/mock"
	"github.com/wrgl/wrgl/pkg/testutils"
)

func (s *testSuite) TestUploadPack(t *testing.T) {
	repo, cli, _, cleanup := s.s.NewClient(t, true, "", nil)
	defer cleanup()
	db := s.s.GetDB(repo)
	rs := s.s.GetRS(repo)
	sum1, c1 := apitest.CreateRandomCommit(t, db, 5, 1000, nil)
	sum2, c2 := apitest.CreateRandomCommit(t, db, 5, 1000, [][]byte{sum1})
	sum3, _ := apitest.CreateRandomCommit(t, db, 5, 1000, nil)
	sum4, _ := apitest.CreateRandomCommit(t, db, 5, 1000, [][]byte{sum3})
	require.NoError(t, ref.CommitHead(rs, "main", sum2, c2))
	require.NoError(t, ref.SaveTag(rs, "v1", sum4))

	dbc := objmock.NewStore()
	rsc := refmock.NewStore()
	commits := apitest.FetchObjects(t, dbc, rsc, cli, [][]byte{sum2}, 0, 0)
	assert.Equal(t, [][]byte{sum1, sum2}, commits)
	apitest.AssertCommitsPersisted(t, db, commits)

	apitest.CopyCommitsToNewStore(t, db, dbc, [][]byte{sum1})
	require.NoError(t, ref.CommitHead(rsc, "main", sum1, c1))
	_, err := apiclient.NewUploadPackSession(db, rs, cli, [][]byte{sum2}, 0, 0, nil)
	assert.Error(t, err, "nothing wanted")

	apitest.CopyCommitsToNewStore(t, db, dbc, [][]byte{sum3})
	require.NoError(t, ref.SaveTag(rsc, "v0", sum3))
	commits = apitest.FetchObjects(t, dbc, rsc, cli, [][]byte{sum2, sum4}, 1, 0)
	apitest.AssertCommitsPersisted(t, db, commits)
}

func (s *testSuite) TestUploadPackMultiplePackfiles(t *testing.T) {
	repo, cli, _, cleanup := s.s.NewClient(t, true, "", nil)
	defer cleanup()
	db := s.s.GetDB(repo)
	rs := s.s.GetRS(repo)
	cs := s.s.GetConfS(repo)
	c, err := cs.Open()
	if err != nil {
		panic(err)
	}
	c.Pack = &conf.Pack{
		MaxFileSize: 1024,
	}
	sum1, _ := apitest.CreateRandomCommit(t, db, 5, 1000, nil)
	sum2, c2 := apitest.CreateRandomCommit(t, db, 5, 1000, [][]byte{sum1})
	require.NoError(t, ref.CommitHead(rs, "main", sum2, c2))

	dbc := objmock.NewStore()
	rsc := refmock.NewStore()
	commits := apitest.FetchObjects(t, dbc, rsc, cli, [][]byte{sum2}, 0, 0)
	assert.Equal(t, [][]byte{sum1, sum2}, commits)
	apitest.AssertCommitsPersisted(t, db, commits)
}

func (s *testSuite) TestUploadPackCustomHeader(t *testing.T) {
	repo, cli, m, cleanup := s.s.NewClient(t, true, "", nil)
	defer cleanup()
	db := s.s.GetDB(repo)
	rs := s.s.GetRS(repo)
	sum1, c1 := apitest.CreateRandomCommit(t, db, 3, 4, nil)
	require.NoError(t, ref.CommitHead(rs, "main", sum1, c1))

	dbc := objmock.NewStore()
	rsc := refmock.NewStore()
	req := m.Capture(t, func(header http.Header) {
		header.Set("Custom-Header", "asd")
		commits := apitest.FetchObjects(t, dbc, rsc, cli, [][]byte{sum1}, 0, 0, apiclient.WithRequestHeader(header))
		assert.Equal(t, [][]byte{sum1}, commits)
		apitest.AssertCommitsPersisted(t, db, commits)
	})
	assert.Equal(t, "asd", req.Header.Get("Custom-Header"))
}

func (s *testSuite) TestUploadPackWithDepth(t *testing.T) {
	repo, cli, _, cleanup := s.s.NewClient(t, true, "", nil)
	defer cleanup()
	db := s.s.GetDB(repo)
	rs := s.s.GetRS(repo)
	sum1, c1 := apitest.CreateRandomCommit(t, db, 3, 4, nil)
	sum2, c2 := apitest.CreateRandomCommit(t, db, 3, 4, [][]byte{sum1})
	sum3, c3 := apitest.CreateRandomCommit(t, db, 3, 4, [][]byte{sum2})
	sum4, c4 := apitest.CreateRandomCommit(t, db, 3, 4, [][]byte{sum3})
	require.NoError(t, ref.CommitHead(rs, "main", sum4, c4))

	dbc := objmock.NewStore()
	rsc := refmock.NewStore()
	commits := apitest.FetchObjects(t, dbc, rsc, cli, [][]byte{sum4}, 0, 2)
	testutils.AssertBytesEqual(t, [][]byte{sum4, sum3, sum2, sum1}, commits, true)
	apitest.AssertCommitsShallowlyPersisted(t, dbc, commits)
	apitest.AssertTablesPersisted(t, dbc, [][]byte{c4.Table, c3.Table})
	apitest.AssertTablesNotPersisted(t, dbc, [][]byte{c2.Table, c1.Table})

	// get missing tables with GetObjects
	pr, err := cli.GetObjects([][]byte{c2.Table, c1.Table})
	require.NoError(t, err)
	defer pr.Close()
	or := apiutils.NewObjectReceiver(dbc, nil, nil)
	done, err := or.Receive(pr, nil)
	require.NoError(t, err)
	assert.True(t, done)
	apitest.AssertTablesPersisted(t, dbc, [][]byte{c2.Table, c1.Table})
}
