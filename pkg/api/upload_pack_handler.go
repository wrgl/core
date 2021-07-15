// SPDX-License-Identifier: Apache-2.0
// Copyright © 2021 Wrangle Ltd

package api

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/wrgl/core/pkg/objects"
	"github.com/wrgl/core/pkg/ref"
)

const PathUploadPack = "/upload-pack/"

type UploadPackHandler struct {
	db              objects.Store
	rs              ref.Store
	sessions        map[string]*UploadPackSession
	maxPackfileSize uint64
}

func NewUploadPackHandler(db objects.Store, rs ref.Store, maxPackfileSize uint64) *UploadPackHandler {
	return &UploadPackHandler{
		db:              db,
		rs:              rs,
		maxPackfileSize: maxPackfileSize,
		sessions:        map[string]*UploadPackSession{},
	}
}

func (h *UploadPackHandler) getSession(r *http.Request) (ses *UploadPackSession, sid string, err error) {
	var ok bool
	c, err := r.Cookie(uploadPackSessionCookie)
	if err == nil {
		sid = c.Value
		ses, ok = h.sessions[sid]
		if !ok {
			ses = nil
		}
	}
	if ses == nil {
		var id uuid.UUID
		id, err = uuid.NewRandom()
		if err != nil {
			return
		}
		sid = id.String()
		ses = NewUploadPackSession(h.db, h.rs, sid, h.maxPackfileSize)
		h.sessions[sid] = ses
	}
	return
}

func (h *UploadPackHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(rw, "forbidden", http.StatusForbidden)
		return
	}
	ses, sid, err := h.getSession(r)
	if err != nil {
		panic(err)
	}
	defer func() {
		if s := recover(); s != nil {
			delete(h.sessions, sid)
			panic(s)
		}
	}()
	if done := ses.ServeHTTP(rw, r); done {
		delete(h.sessions, sid)
	}
}
