// SPDX-License-Identifier: Apache-2.0
// Copyright © 2021 Wrangle Ltd

package authfs

import (
	_ "embed"
	"net/http"
	"os"
	"path/filepath"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	fileadapter "github.com/casbin/casbin/v2/persist/file-adapter"
)

//go:embed casbin_model.conf
var casbinModel string

type AuthzStore struct {
	e *casbin.Enforcer
}

func NewAuthzStore(rootDir string) (s *AuthzStore, err error) {
	m, err := model.NewModelFromString(casbinModel)
	if err != nil {
		return
	}
	fp := filepath.Join(rootDir, "authz.csv")
	if _, err := os.Stat(fp); os.IsNotExist(err) {
		f, err := os.OpenFile(fp, os.O_CREATE, 0600)
		if err != nil {
			return nil, err
		}
		if err := f.Close(); err != nil {
			return nil, err
		}
	}
	e, err := casbin.NewEnforcer(m, fileadapter.NewAdapter(fp))
	if err != nil {
		return
	}
	s = &AuthzStore{
		e: e,
	}
	return s, nil
}

func (s *AuthzStore) AddPolicy(email, act string) error {
	_, err := s.e.AddPolicy(email, "-", act)
	return err
}

func (s *AuthzStore) RemovePolicy(email, act string) error {
	_, err := s.e.RemovePolicy(email, "-", act)
	return err
}

func (s *AuthzStore) Authorized(r *http.Request, email, scope string) (bool, error) {
	return s.e.Enforce(email, "-", scope)
}

func (s *AuthzStore) Flush() error {
	return s.e.SavePolicy()
}

func (s *AuthzStore) ListPolicies(email string) (scopes []string, err error) {
	policies := s.e.GetFilteredPolicy(0, email, "-")
	scopes = make([]string, len(policies))
	for i, sl := range policies {
		scopes[i] = sl[2]
	}
	return scopes, nil
}
