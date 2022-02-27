package apiclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/wrgl/wrgl/pkg/api/payload"
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
	CommitSum []byte
	TableSum  []byte
}

func NewShallowCommitError(comSum, tblSum []byte) *ShallowCommitError {
	return &ShallowCommitError{comSum, tblSum}
}

func (e *ShallowCommitError) Error() string {
	return fmt.Sprintf("commit %x is shallow", e.CommitSum)
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
