// SPDX-License-Identifier: Apache-2.0
// Copyright © 2021 Wrangle Ltd

package apiserver

import (
	"encoding/hex"
	"net/http"
	"regexp"

	"github.com/wrgl/core/pkg/api/payload"
	"github.com/wrgl/core/pkg/objects"
)

var commitURIPat = regexp.MustCompile(`/commits/([0-9a-f]{32})/`)

func (s *Server) handleGetCommit(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(rw, "forbidden", http.StatusForbidden)
		return
	}
	m := commitURIPat.FindStringSubmatch(r.URL.Path)
	if m == nil {
		http.NotFound(rw, r)
		return
	}
	sum, err := hex.DecodeString(m[1])
	if err != nil {
		panic(err)
	}
	repo := getRepo(r)
	db := s.getDB(repo)
	com, err := objects.GetCommit(db, sum)
	if err != nil {
		http.NotFound(rw, r)
		return
	}
	resp := &payload.GetCommitResponse{
		AuthorName:  com.AuthorName,
		AuthorEmail: com.AuthorEmail,
		Message:     com.Message,
		Time:        com.Time,
		Table:       &payload.Hex{},
	}
	copy((*resp.Table)[:], com.Table)
	for _, sum := range com.Parents {
		h := &payload.Hex{}
		copy((*h)[:], sum)
		resp.Parents = append(resp.Parents, h)
	}
	writeJSON(rw, resp)
}
