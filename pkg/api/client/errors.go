// SPDX-License-Identifier: Apache-2.0
// Copyright © 2022 Wrangle Ltd

package apiclient

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/wrgl/wrgl/pkg/api/payload"
	"github.com/wrgl/wrgl/pkg/objects"
	"github.com/wrgl/wrgl/pkg/ref"
)

type HTTPError struct {
	Code    int
	RawBody []byte
	Body    *payload.Error
}

func NewHTTPError(resp *http.Response) *HTTPError {
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	obj := &HTTPError{
		Code:    resp.StatusCode,
		RawBody: b,
	}
	if s := resp.Header.Get("Content-Type"); s == CTJSON {
		obj.Body = &payload.Error{}
		if err := json.Unmarshal(b, obj.Body); err != nil {
			panic(err)
		}
	} else {
		obj.RawBody = b
	}
	return obj
}

func (obj *HTTPError) Error() string {
	b := obj.RawBody
	var err error
	if obj.Body != nil {
		b, err = json.Marshal(obj.Body)
		if err != nil {
			panic(err)
		}
	}
	return fmt.Sprintf("status %d: %s", obj.Code, strings.TrimSpace(string(b)))
}

type ShallowCommitError struct {
	CommitSums [][]byte
	TableSums  map[string][][]byte
}

func NewShallowCommitError(db objects.Store, rs ref.Store, coms []*objects.Commit) error {
	e := &ShallowCommitError{
		TableSums: map[string][][]byte{},
	}
	for _, com := range coms {
		if !objects.TableExist(db, com.Table) {
			e.CommitSums = append(e.CommitSums, com.Sum)
			rem, err := FindRemoteFor(db, rs, com.Sum)
			if err != nil {
				return err
			}
			if rem == "" {
				return fmt.Errorf("no remote found for table %x", com.Table)
			}
			e.TableSums[rem] = append(e.TableSums[rem], com.Table)
		}
	}
	if len(e.CommitSums) > 0 {
		return e
	}
	return nil
}

func (e *ShallowCommitError) Error() string {
	comSums := make([]string, len(e.CommitSums))
	for i, v := range e.CommitSums {
		comSums[i] = hex.EncodeToString(v)
	}
	cmds := make([]string, 0, len(e.TableSums))
	for rem, sl := range e.TableSums {
		tblSums := make([]string, len(sl))
		for i, v := range sl {
			tblSums[i] = hex.EncodeToString(v)
		}
		cmds = append(cmds, fmt.Sprintf("wrgl fetch tables %s %s", rem, strings.Join(tblSums, " ")))
	}
	if len(comSums) == 1 {
		return fmt.Sprintf(
			"commit %s is shallow\nrun this command to fetch their content:\n  %s",
			strings.Join(comSums, ", "),
			strings.Join(cmds, "\n  "),
		)
	}
	return fmt.Sprintf(
		"commits %s are shallow\nrun this command to fetch their content:\n  %s",
		strings.Join(comSums, ", "),
		strings.Join(cmds, "\n  "),
	)
}

func UnwrapHTTPError(err error) *HTTPError {
	werr := err
	for {
		if v, ok := werr.(*HTTPError); ok {
			return v
		}
		werr = errors.Unwrap(werr)
		if werr == nil {
			return nil
		}
	}
}
