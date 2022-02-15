// SPDX-License-Identifier: Apache-2.0
// Copyright © 2021 Wrangle Ltd

package wrgld

import (
	"context"
	"log"
	"net/http"
	"time"

	authlocal "github.com/wrgl/wrgl/cmd/wrgld/auth/local"
	authoidc "github.com/wrgl/wrgl/cmd/wrgld/auth/oidc"
	wrgldutils "github.com/wrgl/wrgl/cmd/wrgld/utils"
	apiserver "github.com/wrgl/wrgl/pkg/api/server"
	authfs "github.com/wrgl/wrgl/pkg/auth/fs"
	"github.com/wrgl/wrgl/pkg/conf"
	conffs "github.com/wrgl/wrgl/pkg/conf/fs"
	"github.com/wrgl/wrgl/pkg/local"
	"github.com/wrgl/wrgl/pkg/objects"
	"github.com/wrgl/wrgl/pkg/ref"
)

type ServerOptions struct {
	ObjectsStore objects.Store

	RefStore ref.Store

	ConfigStore conf.Store

	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type Server struct {
	srv      *http.Server
	cleanups []func()
}

func NewServer(rd *local.RepoDir, readTimeout, writeTimeout time.Duration) (*Server, error) {
	objstore, err := rd.OpenObjectsStore()
	if err != nil {
		return nil, err
	}
	refstore := rd.OpenRefStore()
	cs := conffs.NewStore(rd.FullPath, conffs.AggregateSource, "")
	c, err := cs.Open()
	if err != nil {
		return nil, err
	}
	upSessions := apiserver.NewUploadPackSessionMap()
	rpSessions := apiserver.NewReceivePackSessionMap()
	s := &Server{
		srv: &http.Server{
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
		},
		cleanups: []func(){
			func() { rd.Close() },
			func() { objstore.Close() },
		},
	}
	var handler http.Handler = apiserver.NewServer(
		nil,
		func(r *http.Request) objects.Store { return objstore },
		func(r *http.Request) ref.Store { return refstore },
		func(r *http.Request) conf.Store { return cs },
		func(r *http.Request) apiserver.UploadPackSessionStore { return upSessions },
		func(r *http.Request) apiserver.ReceivePackSessionStore { return rpSessions },
	)
	if c.Auth.Type == conf.ATLegacy {
		authnS, err := authfs.NewAuthnStore(rd, c.TokenDuration())
		if err != nil {
			return nil, err
		}
		authzS, err := authfs.NewAuthzStore(rd)
		if err != nil {
			return nil, err
		}
		handler = authlocal.NewHandler(handler, authnS, authzS)
		s.cleanups = append(s.cleanups,
			func() { authnS.Close() },
			func() { authzS.Close() },
		)
	} else {
		handler, err = authoidc.NewHandler(handler, c.Auth)
		if err != nil {
			return nil, err
		}
	}
	s.srv.Handler = wrgldutils.ApplyMiddlewares(
		handler,
		LoggingMiddleware,
		RecoveryMiddleware,
	)
	return s, nil
}

func (s *Server) Start(addr string) error {
	s.srv.Addr = addr
	log.Printf("server started at %s", addr)
	return s.srv.ListenAndServe()
}

func (s *Server) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.srv.Shutdown(ctx); err != nil {
		return err
	}
	for i := len(s.cleanups) - 1; i >= 0; i-- {
		s.cleanups[i]()
	}
	return nil
}
