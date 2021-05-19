// SPDX-License-Identifier: Apache-2.0
// Copyright © 2021 Wrangle Ltd

package packclient

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"strings"

	"github.com/wrgl/core/pkg/encoding"
	"github.com/wrgl/core/pkg/misc"
	"golang.org/x/net/publicsuffix"
)

type Client struct {
	client *http.Client
	// origin is the scheme + host name of remote server
	origin string
}

func NewClient(origin string) (*Client, error) {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, err
	}
	return &Client{
		client: &http.Client{
			Jar: jar,
		},
		origin: origin,
	}, nil
}

func (c *Client) request(method, path string, body io.Reader, headers map[string]string) (resp *http.Response, err error) {
	req, err := http.NewRequest(method, c.origin+path, body)
	if err != nil {
		return
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err = c.client.Do(req)
	if err != nil {
		return
	}
	if resp.StatusCode >= 400 {
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("%s: %s", resp.Status, string(b))
	}
	return
}

func (c *Client) GetRefsInfo() (m map[string][]byte, err error) {
	resp, err := c.request(http.MethodGet, "/info/refs/", nil, nil)
	if err != nil {
		return
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/x-wrgl-upload-pack-advertisement" {
		return nil, fmt.Errorf("unrecognized content type: %q", ct)
	}
	defer resp.Body.Close()
	m = map[string][]byte{}
	parser := encoding.NewParser(resp.Body)
	var s string
	for {
		s, err = encoding.ReadPktLine(parser)
		if err != nil {
			return
		}
		if s == "" {
			break
		}
		b := make([]byte, 16)
		_, err = hex.Decode(b, []byte(s[:32]))
		if err != nil {
			return
		}
		m[s[33:]] = b
	}
	return
}

func (c *Client) sendUploadPackRequest(wants, haves [][]byte, done bool) (resp *http.Response, err error) {
	body := bytes.NewBuffer(nil)
	buf := misc.NewBuffer(nil)
	for _, want := range wants {
		err = encoding.WritePktLine(body, buf, "want "+hex.EncodeToString(want))
		if err != nil {
			return
		}
	}
	for _, have := range haves {
		err = encoding.WritePktLine(body, buf, "have "+hex.EncodeToString(have))
		if err != nil {
			return
		}
	}
	if done {
		err = encoding.WritePktLine(body, buf, "done")
	} else {
		err = encoding.WritePktLine(body, buf, "")
	}
	if err != nil {
		return
	}
	return c.request(http.MethodPost, "/upload-pack/", body, map[string]string{
		"Content-Type": "application/x-wrgl-upload-pack-request",
	})
}

func parseUploadPackResult(r io.ReadCloser) (acks [][]byte, err error) {
	defer r.Close()
	parser := encoding.NewParser(r)
	var s string
	for {
		s, err = encoding.ReadPktLine(parser)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(s, "ACK ") {
			b := make([]byte, 16)
			_, err = hex.Decode(b, []byte(s[4:]))
			if err != nil {
				return nil, err
			}
			acks = append(acks, b)
		} else if strings.HasPrefix(s, "NAK") {
			break
		}
	}
	return
}

func (c *Client) PostUploadPack(wants, haves [][]byte, done bool) (acks [][]byte, pr *encoding.PackfileReader, err error) {
	resp, err := c.sendUploadPackRequest(wants, haves, done)
	if err != nil {
		return
	}
	if resp.Header.Get("Content-Type") == "application/x-wrgl-packfile" {
		pr, err = encoding.NewPackfileReader(resp.Body)
	} else if resp.Header.Get("Content-Type") == "application/x-wrgl-upload-pack-result" {
		acks, err = parseUploadPackResult(resp.Body)
	}
	return
}
