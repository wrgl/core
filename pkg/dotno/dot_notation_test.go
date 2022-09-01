// SPDX-License-Identifier: Apache-2.0
// Copyright © 2022 Wrangle Ltd

package dotno

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wrgl/wrgl/pkg/conf"
)

func TestParseSliceIndex(t *testing.T) {
	sl := []string{"a", "b", "c"}
	v := reflect.ValueOf(sl)
	_, err := parseSliceIndex(v, "a")
	assert.Error(t, err)
	_, err = parseSliceIndex(v, "10")
	assert.Error(t, err)
	i, err := parseSliceIndex(v, "2")
	require.NoError(t, err)
	assert.Equal(t, 2, i)
}

func TestGetFieldValue(t *testing.T) {
	for i, c := range []struct {
		Obj           interface{}
		Prop          string
		CreateIfZero  bool
		ExpectedValue interface{}
		ExpectedErr   error
	}{
		{
			Obj:           "abc",
			ExpectedValue: "abc",
		},
		{
			Obj: &conf.Auth{
				AnonymousRead: true,
			},
			Prop:          "anonymousRead",
			ExpectedValue: true,
		},
		{
			Obj: &conf.Auth{
				AnonymousRead: true,
			},
			Prop:        "abc",
			ExpectedErr: fmt.Errorf(`field "Abc" not found`),
		},
		{
			Obj: &conf.Config{
				Auth: &conf.Auth{
					AnonymousRead: true,
				},
			},
			Prop:          "auth.anonymousRead",
			ExpectedValue: true,
		},
		{
			Obj:         &conf.Config{},
			Prop:        "auth.anonymousRead",
			ExpectedErr: fmt.Errorf(`field "Auth" is zero`),
		},
		{
			Obj:           &conf.Config{},
			Prop:          "auth",
			CreateIfZero:  true,
			ExpectedValue: &conf.Auth{},
		},
		{
			Obj:           &conf.Config{},
			Prop:          "auth.repositoryName",
			CreateIfZero:  true,
			ExpectedValue: "",
		},
		{
			Obj:           &conf.Config{},
			Prop:          "remote.origin",
			CreateIfZero:  true,
			ExpectedValue: &conf.Remote{},
		},
		{
			Obj:         &conf.Config{Remote: map[string]*conf.Remote{}},
			Prop:        "remote.origin.mirror",
			ExpectedErr: fmt.Errorf(`key not found: "origin"`),
		},
		{
			Obj:           &conf.Config{Remote: map[string]*conf.Remote{}},
			Prop:          "remote.origin.mirror",
			CreateIfZero:  true,
			ExpectedValue: false,
		},
		{
			Obj: &conf.Branch{
				PrimaryKey: []string{},
			},
			Prop:        "primaryKey.0",
			ExpectedErr: fmt.Errorf("index out of range: 0 >= 0"),
		},
	} {
		v, err := GetFieldValue(c.Obj, c.Prop, c.CreateIfZero)
		if err != nil {
			assert.Equal(t, c.ExpectedErr, err, "case %d", i)
		} else {
			assert.Equal(t, c.ExpectedValue, v.Interface(), "case %d", i)
		}
	}
}

func TestGetParentField(t *testing.T) {
	c := &conf.Config{
		Remote: map[string]*conf.Remote{
			"origin": {
				URL: "http://my-remote.com",
			},
		},
	}
	parent, name, err := GetParentField(c, "remote.origin")
	require.NoError(t, err)
	assert.Equal(t, c.Remote, parent.Interface())
	assert.Equal(t, "origin", name)

	parent, name, err = GetParentField(c, "remote.origin.url")
	require.NoError(t, err)
	assert.Equal(t, c.Remote["origin"], parent.Interface())
	assert.Equal(t, "url", name)
}

func duration(v int64) *conf.Duration {
	c := conf.Duration(v)
	return &c
}

func TestSetValue(t *testing.T) {
	for i, c := range []struct {
		Obj         interface{}
		Prop        string
		Value       string
		ExpectedObj interface{}
		ExpectedErr error
	}{
		{
			Obj:   &conf.Auth{},
			Prop:  "repositoryName",
			Value: "my repo",
			ExpectedObj: &conf.Auth{
				RepositoryName: "my repo",
			},
		},
		{
			Obj:   &conf.Config{},
			Prop:  "auth.repositoryName",
			Value: "my repo",
			ExpectedObj: &conf.Config{
				Auth: &conf.Auth{
					RepositoryName: "my repo",
				},
			},
		},
		{
			Obj:   &conf.Pack{},
			Prop:  "maxFileSize",
			Value: "123456",
			ExpectedObj: &conf.Pack{
				MaxFileSize: 123456,
			},
		},
		{
			Obj:   &conf.Auth{},
			Prop:  "keycloak",
			Value: `{"issuer": "http://keycloak"}`,
			ExpectedObj: &conf.Auth{
				Keycloak: &conf.AuthKeycloak{
					Issuer: "http://keycloak",
				},
			},
		},
		{
			Obj:   &conf.Branch{},
			Prop:  "primaryKey",
			Value: "uid,uof_uid",
			ExpectedObj: &conf.Branch{
				PrimaryKey: []string{"uid", "uof_uid"},
			},
		},
	} {
		v, err := GetFieldValue(c.Obj, c.Prop, true)
		if err != nil {
			assert.Equal(t, c.ExpectedErr, err, "case %d", i)
		} else {
			err = SetValue(v, c.Value)
			if err != nil {
				assert.Equal(t, c.ExpectedErr, err, "case %d", i)
			} else {
				assert.Equal(t, c.ExpectedObj, c.Obj)
			}
		}
	}
}

func TestUnsetField(t *testing.T) {
	for i, c := range []struct {
		Obj         interface{}
		Prop        string
		All         bool
		ExpectedObj interface{}
		ExpectedErr error
	}{
		{
			Obj: &conf.Auth{
				AnonymousRead:  true,
				RepositoryName: "my repo",
			},
			Prop: "anonymousRead",
			ExpectedObj: &conf.Auth{
				RepositoryName: "my repo",
			},
		},
		{
			Obj: &conf.Branch{
				PrimaryKey: []string{"abc123"},
			},
			Prop:        "primaryKey",
			ExpectedObj: &conf.Branch{},
		},
		{
			Obj: &conf.Branch{
				PrimaryKey: []string{"abc123", "def456"},
			},
			Prop:        "primaryKey",
			ExpectedErr: fmt.Errorf("key contains multiple values"),
		},
		{
			Obj: &conf.Branch{
				PrimaryKey: []string{"abc123", "def456"},
			},
			Prop:        "primaryKey",
			All:         true,
			ExpectedObj: &conf.Branch{},
		},
		{
			Obj: map[string]*conf.Remote{
				"origin": {
					URL: "http://my-remote.com",
				},
				"origin2": {
					URL: "http://my-remote2.com",
				},
			},
			Prop: "origin",
			ExpectedObj: map[string]*conf.Remote{
				"origin2": {
					URL: "http://my-remote2.com",
				},
			},
		},
		{
			Obj: &conf.Branch{
				PrimaryKey: []string{"123", "456", "789"},
			},
			Prop: "primaryKey.1",
			ExpectedObj: &conf.Branch{
				PrimaryKey: []string{"123", "789"},
			},
		},
		{
			Obj: &conf.Branch{
				PrimaryKey: []string{"123", "456", "789"},
			},
			Prop: "primaryKey.0",
			ExpectedObj: &conf.Branch{
				PrimaryKey: []string{"456", "789"},
			},
		},
		{
			Obj: &conf.Branch{
				PrimaryKey: []string{"123", "456", "789"},
			},
			Prop: "primaryKey.2",
			ExpectedObj: &conf.Branch{
				PrimaryKey: []string{"123", "456"},
			},
		},
		{
			Obj: &conf.Branch{
				PrimaryKey: []string{"123", "456", "789"},
			},
			Prop:        "primaryKey.10",
			ExpectedErr: fmt.Errorf("index out of range: 10 >= 3"),
		},
	} {
		err := UnsetField(c.Obj, c.Prop, c.All)
		if err != nil {
			assert.Equal(t, c.ExpectedErr, err, "case %d", i)
		} else {
			assert.Equal(t, c.ExpectedObj, c.Obj, "case %d", i)
		}
	}
}

func TestAppendSlice(t *testing.T) {
	branch := &conf.Branch{
		PrimaryKey: []string{"a", "b"},
	}
	v := reflect.ValueOf(branch).Elem().FieldByName("PrimaryKey")
	assert.Equal(t, fmt.Errorf("can only append to pointer of slice"), AppendSlice(v, "c"))
	v = reflect.ValueOf(branch).Elem().FieldByName("PrimaryKey").Addr()
	require.NoError(t, AppendSlice(v, "c", "d"))
	assert.Equal(t, []string{"a", "b", "c", "d"}, branch.PrimaryKey)

	rem := &conf.Remote{}
	v = reflect.ValueOf(rem).Elem().FieldByName("Fetch").Addr()
	require.NoError(t, AppendSlice(v, "refs/remotes/origin/main:refs/heads/main"))
	assert.Equal(t, conf.RefspecSlice{
		conf.MustParseRefspec("refs/remotes/origin/main:refs/heads/main"),
	}, rem.Fetch)
}
