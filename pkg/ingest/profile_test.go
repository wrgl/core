// SPDX-License-Identifier: Apache-2.0
// Copyright © 2022 Wrangle Ltd

package ingest

import (
	"os"
	"testing"

	"github.com/go-logr/logr/testr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wrgl/wrgl/pkg/objects"
	objmock "github.com/wrgl/wrgl/pkg/objects/mock"
	"github.com/wrgl/wrgl/pkg/sorter"
	"github.com/wrgl/wrgl/pkg/testutils"
)

func TestProfileTable(t *testing.T) {
	rows := testutils.BuildRawCSV(4, 700)
	f := writeCSV(t, rows)
	defer os.Remove(f.Name())
	db := objmock.NewStore()
	s, err := sorter.NewSorter()
	require.NoError(t, err)
	logger := testr.New(t)

	sum, err := IngestTable(db, s, f, rows[0][:1], logger)
	require.NoError(t, err)
	tbl, err := objects.GetTable(db, sum)
	require.NoError(t, err)
	tblProf, err := objects.GetTableProfile(db, sum)
	require.NoError(t, err)
	require.NoError(t, objects.DeleteTableProfile(db, sum))

	require.NoError(t, ProfileTable(db, sum, tbl))
	tblSum2, err := objects.GetTableProfile(db, sum)
	require.NoError(t, err)
	assert.Equal(t, tblProf, tblSum2)
}
