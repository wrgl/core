// SPDX-License-Identifier: Apache-2.0
// Copyright © 2021 Wrangle Ltd

package apiserver

import (
	"encoding/hex"
	"net/http"
	"regexp"

	"github.com/wrgl/core/pkg/api/payload"
	"github.com/wrgl/core/pkg/diff"
	"github.com/wrgl/core/pkg/objects"
)

var diffURIPat = regexp.MustCompile(`/diff/([0-9a-f]{32})/([0-9a-f]{32})/`)

func (s *Server) getTable(db objects.Store, x string) ([]byte, *objects.Table, [][]string) {
	sum, err := hex.DecodeString(x)
	if err != nil {
		panic(err)
	}
	com, err := objects.GetCommit(db, sum)
	if err != nil {
		return nil, nil, nil
	}
	tbl, err := objects.GetTable(db, com.Table)
	if err != nil {
		panic(err)
	}
	idx, err := objects.GetTableIndex(db, com.Table)
	if err != nil {
		panic(err)
	}
	return com.Table, tbl, idx
}

func (s *Server) handleDiff(rw http.ResponseWriter, r *http.Request) {
	m := diffURIPat.FindStringSubmatch(r.URL.Path)
	if m == nil {
		sendHTTPError(rw, http.StatusNotFound)
		return
	}
	db := s.getDB(r)
	sum1, tbl1, idx1 := s.getTable(db, m[1])
	if tbl1 == nil {
		sendHTTPError(rw, http.StatusNotFound)
		return
	}
	sum2, tbl2, idx2 := s.getTable(db, m[2])
	if tbl2 == nil {
		sendHTTPError(rw, http.StatusNotFound)
		return
	}
	errCh := make(chan error, 10)
	diffChan, _ := diff.DiffTables(db, db, tbl1, tbl2, idx1, idx2, 0, errCh, false)
	resp := &payload.DiffResponse{
		TableSum:    payload.BytesToHex(sum1),
		OldTableSum: payload.BytesToHex(sum2),
		ColDiff: &payload.ColDiff{
			Columns:    tbl1.Columns,
			OldColumns: tbl2.Columns,
			PK:         tbl1.PK,
			OldPK:      tbl2.PK,
		},
	}
	for obj := range diffChan {
		resp.RowDiff = append(resp.RowDiff, &payload.RowDiff{
			PK:        payload.BytesToHex(obj.PK),
			Sum:       payload.BytesToHex(obj.Sum),
			OldSum:    payload.BytesToHex(obj.OldSum),
			Offset:    obj.Offset,
			OldOffset: obj.OldOffset,
		})
	}
	close(errCh)
	err, ok := <-errCh
	if ok {
		panic(err)
	}
	s.cacheControlImmutable(rw)
	writeJSON(rw, resp)
}
