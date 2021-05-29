// SPDX-License-Identifier: Apache-2.0
// Copyright © 2021 Wrangle Ltd

package diff

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wrgl/core/pkg/ingest"
	"github.com/wrgl/core/pkg/kv"
	"github.com/wrgl/core/pkg/objects"
	"github.com/wrgl/core/pkg/table"
	"github.com/wrgl/core/pkg/testutils"
)

func TestDiffTables(t *testing.T) {
	cases := []struct {
		T1     table.Store
		T2     table.Store
		Events []objects.Diff
	}{
		{
			table.NewMockStore([]string{"one", "two"}, []uint32{0}, nil),
			table.NewMockStore([]string{"one", "three"}, []uint32{1}, nil),
			[]objects.Diff{
				{Type: objects.DTColumnAdd, Columns: []string{"two"}},
				{Type: objects.DTColumnRem, Columns: []string{"three"}},
				{Type: objects.DTPKChange, Columns: []string{"three"}},
			},
		},
		{
			table.NewMockStore([]string{"one", "two"}, []uint32{0}, nil),
			table.NewMockStore([]string{"one", "two"}, []uint32{0}, nil),
			[]objects.Diff{},
		},
		{
			table.NewMockStore([]string{"one", "two"}, []uint32{0}, nil),
			table.NewMockStore([]string{"one", "two"}, []uint32{}, nil),
			[]objects.Diff{
				{Type: objects.DTPKChange, Columns: []string{}},
			},
		},
		{
			table.NewMockStore([]string{"a", "b", "c", "d"}, []uint32{0, 2}, nil),
			table.NewMockStore([]string{"a", "c", "d", "b"}, []uint32{0, 1}, nil),
			[]objects.Diff{},
		},
		{
			table.NewMockStore([]string{"a", "b", "c", "d"}, []uint32{0, 1}, nil),
			table.NewMockStore([]string{"b", "a", "c", "d"}, []uint32{0, 1}, nil),
			[]objects.Diff{
				{Type: objects.DTPKChange, Columns: []string{"b", "a"}},
			},
		},
		{
			table.NewMockStore([]string{"a", "b"}, nil, [][2]string{
				{"abc", "123"},
				{"def", "456"},
				{"qwe", "234"},
			}),
			table.NewMockStore([]string{"a", "b", "c"}, nil, [][2]string{
				{"abc", "123"},
				{"def", "059"},
				{"asd", "789"},
			}),
			[]objects.Diff{
				{Type: objects.DTColumnRem, Columns: []string{"c"}},
			},
		},
		{
			table.NewMockStore([]string{"a", "b"}, nil, [][2]string{
				{"abc", "123"},
				{"def", "456"},
				{"qwe", "234"},
			}),
			table.NewMockStore([]string{"a", "b"}, nil, [][2]string{
				{"abc", "123"},
				{"def", "059"},
				{"asd", "789"},
			}),
			[]objects.Diff{
				{Type: objects.DTRow, PK: []byte("def"), Sum: []byte("456"), OldSum: []byte("059")},
				{Type: objects.DTRow, PK: []byte("qwe"), Sum: []byte("234")},
				{Type: objects.DTRow, PK: []byte("asd"), OldSum: []byte("789")},
			},
		},
		{
			table.NewMockStore([]string{"a", "b"}, nil, [][2]string{
				{"abc", "123"},
				{"def", "456"},
				{"qwe", "234"},
			}),
			table.NewMockStore([]string{"a", "b"}, nil, [][2]string{
				{"abc", "123"},
				{"def", "456"},
				{"qwe", "234"},
			}),
			[]objects.Diff{},
		},
		{
			table.NewMockStore([]string{"a", "b"}, nil, [][2]string{
				{"abc", "123"},
				{"def", "456"},
				{"qwe", "234"},
			}),
			table.NewMockStore([]string{"a", "c"}, nil, [][2]string{
				{"abc", "123"},
				{"def", "456"},
				{"qwe", "234"},
			}),
			[]objects.Diff{
				{Type: objects.DTColumnAdd, Columns: []string{"b"}},
				{Type: objects.DTColumnRem, Columns: []string{"c"}},
			},
		},

		{
			table.NewMockStore([]string{"one", "two"}, []uint32{0}, [][2]string{
				{"abc", "123"},
				{"def", "456"},
				{"qwe", "234"},
			}),
			table.NewMockStore([]string{"one", "two"}, []uint32{0}, [][2]string{
				{"abc", "123"},
				{"def", "059"},
				{"asd", "789"},
			}),
			[]objects.Diff{
				{Type: objects.DTRow, PK: []byte("def"), Sum: []byte("456"), OldSum: []byte("059")},
				{Type: objects.DTRow, PK: []byte("qwe"), Sum: []byte("234")},
				{Type: objects.DTRow, PK: []byte("asd"), OldSum: []byte("789")},
			},
		},
		{
			table.NewMockStore([]string{"one", "two", "three"}, []uint32{0}, [][2]string{
				{"abc", "123"},
				{"def", "456"},
			}),
			table.NewMockStore([]string{"one", "two"}, []uint32{0}, [][2]string{
				{"abc", "345"},
				{"def", "678"},
			}),
			[]objects.Diff{
				{Type: objects.DTColumnAdd, Columns: []string{"three"}},
				{Type: objects.DTRow, PK: []byte("abc"), Sum: []byte("123"), OldSum: []byte("345")},
				{Type: objects.DTRow, PK: []byte("def"), Sum: []byte("456"), OldSum: []byte("678")},
			},
		},
		{
			table.NewMockStore([]string{"one", "two"}, []uint32{0}, [][2]string{
				{"abc", "123"},
				{"def", "456"},
				{"qwe", "234"},
			}),
			table.NewMockStore([]string{"one", "two"}, []uint32{0}, [][2]string{
				{"abc", "123"},
				{"def", "456"},
				{"qwe", "234"},
			}),
			[]objects.Diff{},
		},
	}
	for i, c := range cases {
		errChan := make(chan error, 1000)
		diffChan, _ := DiffTables(c.T1, c.T2, 0, errChan)
		events := []objects.Diff{}
		for e := range diffChan {
			events = append(events, e)
		}
		assert.Equal(t, c.Events, events, "case %d", i)
		close(errChan)
		_, ok := <-errChan
		assert.False(t, ok)
	}
}

func ingestRawCSV(b *testing.B, db kv.DB, fs kv.FileStore, rows [][]string) table.Store {
	b.Helper()
	cols := rows[0]
	reader := testutils.RawCSVReader(rows[1:])
	tb := table.NewBuilder(db, fs, cols, nil, 0, 0)
	sum, err := ingest.Ingest(0, 1, reader, []uint32{}, tb, io.Discard)
	require.NoError(b, err)
	ts, err := table.ReadTable(db, fs, sum)
	require.NoError(b, err)
	return ts
}

func BenchmarkDiffRows(b *testing.B) {
	rawCSV1 := testutils.BuildRawCSV(12, b.N)
	rawCSV2 := testutils.ModifiedCSV(rawCSV1, 1)
	db := kv.NewMockStore(false)
	fs := kv.NewMockStore(false)
	store1 := ingestRawCSV(b, db, fs, rawCSV1)
	store2 := ingestRawCSV(b, db, fs, rawCSV2)
	errChan := make(chan error, 1000)
	b.ResetTimer()
	diffChan, _ := DiffTables(store1, store2, 0, errChan)
	for d := range diffChan {
		assert.NotNil(b, d)
	}
	_, ok := <-errChan
	assert.False(b, ok)
}
