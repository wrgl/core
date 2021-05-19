// SPDX-License-Identifier: Apache-2.0
// Copyright © 2021 Wrangle Ltd

package versioning

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepoDirInit(t *testing.T) {
	dir, err := ioutil.TempDir("", "test_repo_dir")
	require.NoError(t, err)
	defer os.RemoveAll(dir)
	wrglDir := filepath.Join(dir, ".wrgl")
	rd := NewRepoDir(wrglDir, false, false)
	assert.Equal(t, wrglDir, rd.FullPath)
	assert.False(t, rd.Exist())
	err = rd.Init()
	require.NoError(t, err)
	assert.True(t, rd.Exist())

	_, err = os.Stat(rd.KVPath())
	require.NoError(t, err)
	kvs, err := rd.OpenKVStore()
	require.NoError(t, err)
	defer kvs.Close()

	_, err = os.Stat(rd.FilesPath())
	require.NoError(t, err)
	fs := rd.OpenFileStore()
	key := []byte("my-file")
	w, err := fs.Writer(key)
	require.NoError(t, err)
	defer w.Close()
	content := []byte("abc123")
	w.Write(content)
	w.Close()
	r, err := fs.Reader(key)
	require.NoError(t, err)
	defer r.Close()
	b, err := ioutil.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, content, b)
}
