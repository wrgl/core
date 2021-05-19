// SPDX-License-Identifier: Apache-2.0
// Copyright © 2021 Wrangle Ltd

package versioning

import (
	"os"
	"path/filepath"

	"github.com/dgraph-io/badger/v3"
	"github.com/wrgl/core/pkg/kv"
)

type RepoDir struct {
	FullPath       string
	badgerLogInfo  bool
	badgerLogDebug bool
}

func NewRepoDir(wrglDir string, badgerLogInfo, badgerLogDebug bool) *RepoDir {
	return &RepoDir{
		FullPath:       wrglDir,
		badgerLogInfo:  badgerLogInfo,
		badgerLogDebug: badgerLogDebug,
	}
}

func (d *RepoDir) FilesPath() string {
	return filepath.Join(d.FullPath, "files")
}

func (d *RepoDir) KVPath() string {
	return filepath.Join(d.FullPath, "kv")
}

func (d *RepoDir) OpenKVStore() (kv.Store, error) {
	opts := badger.DefaultOptions(d.KVPath()).
		WithLoggingLevel(badger.ERROR)
	if d.badgerLogDebug {
		opts = opts.WithLoggingLevel(badger.DEBUG)
	} else if d.badgerLogInfo {
		opts = opts.WithLoggingLevel(badger.INFO)
	}
	badgerDB, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}
	return kv.NewBadgerStore(badgerDB), nil
}

func (d *RepoDir) OpenFileStore() kv.FileStore {
	return kv.NewFileStore(d.FilesPath())
}

func (d *RepoDir) Init() error {
	err := os.Mkdir(d.FullPath, 0755)
	if err != nil {
		return err
	}
	err = os.Mkdir(d.FilesPath(), 0755)
	if err != nil {
		return err
	}
	return os.Mkdir(d.KVPath(), 0755)
}

func (d *RepoDir) Exist() bool {
	_, err := os.Stat(d.FullPath)
	return err == nil
}
