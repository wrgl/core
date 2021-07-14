// SPDX-License-Identifier: Apache-2.0
// Copyright © 2021 Wrangle Ltd

package pack

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/wrgl/core/pkg/conf"
	"github.com/wrgl/core/pkg/objects"
	"github.com/wrgl/core/pkg/ref"
)

var zeroOID = strings.Repeat("0", 32)

type ReceivePackHandler struct {
	db       objects.Store
	rs       ref.Store
	c        *conf.Config
	Path     string
	sessions map[string]*ReceivePackSession
}

func NewReceivePackHandler(db objects.Store, rs ref.Store, c *conf.Config) *ReceivePackHandler {
	return &ReceivePackHandler{
		db:       db,
		rs:       rs,
		c:        c,
		Path:     "/receive-pack/",
		sessions: map[string]*ReceivePackSession{},
	}
}

func (h *ReceivePackHandler) getSession(r *http.Request) (ses *ReceivePackSession, sid string, err error) {
	var ok bool
	c, err := r.Cookie(receivePackSessionCookie)
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
		ses = NewReceivePackSession(h.db, h.rs, h.c, h.Path, sid)
		h.sessions[sid] = ses
	}
	return
}

func (h *ReceivePackHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
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
