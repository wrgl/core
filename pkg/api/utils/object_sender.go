// SPDX-License-Identifier: Apache-2.0
// Copyright © 2021 Wrangle Ltd

package apiutils

import (
	"bytes"
	"container/list"
	"io"

	"github.com/wrgl/wrgl/pkg/encoding/packfile"
	"github.com/wrgl/wrgl/pkg/objects"
)

const defaultMaxPackfileSize uint64 = 1024 * 1024 * 1024 * 2

type object struct {
	Type    int
	Content []byte
	Sum     []byte
}

type ObjectSender struct {
	db              objects.Store
	commits         *list.List
	tables          *list.List
	objs            *list.List
	commonTables    map[string]struct{}
	commonBlocks    map[string]struct{}
	maxPackfileSize uint64
	buf             *bytes.Buffer
}

func getCommonTables(db objects.Store, commonCommits [][]byte) (map[string]struct{}, error) {
	commonTables := map[string]struct{}{}
	for _, b := range commonCommits {
		c, err := objects.GetCommit(db, b)
		if err != nil {
			return nil, err
		}
		commonTables[string(c.Table)] = struct{}{}
	}
	return commonTables, nil
}

func getCommonBlocks(db objects.Store, commonTables map[string]struct{}) (map[string]struct{}, error) {
	commonBlocks := map[string]struct{}{}
	for b := range commonTables {
		t, err := objects.GetTable(db, []byte(b))
		if err == objects.ErrKeyNotFound {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, blk := range t.Blocks {
			commonBlocks[string(blk)] = struct{}{}
		}
	}
	return commonBlocks, nil
}

func NewObjectSender(db objects.Store, toSend []*objects.Commit, tablesToSend [][]byte, commonCommits [][]byte, maxPackfileSize uint64) (s *ObjectSender, err error) {
	if maxPackfileSize == 0 {
		maxPackfileSize = defaultMaxPackfileSize
	}
	s = &ObjectSender{
		db:              db,
		commits:         list.New(),
		tables:          list.New(),
		objs:            list.New(),
		buf:             bytes.NewBuffer(nil),
		maxPackfileSize: maxPackfileSize,
	}
	for _, com := range toSend {
		s.commits.PushBack(com)
	}
	for _, sum := range tablesToSend {
		s.tables.PushBack(sum)
	}
	s.commonTables, err = getCommonTables(db, commonCommits)
	if err != nil {
		return nil, err
	}
	s.commonBlocks, err = getCommonBlocks(db, s.commonTables)
	if err != nil {
		return nil, err
	}
	err = s.enqueueFrontTable()
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *ObjectSender) enqueueFrontTable() (err error) {
	var sum []byte
	for s.tables.Len() > 0 {
		sum = s.tables.Remove(s.tables.Front()).([]byte)
		if _, ok := s.commonTables[string(sum)]; !ok {
			s.commonTables[string(sum)] = struct{}{}
			break
		} else {
			sum = nil
		}
	}
	if sum == nil {
		return nil
	}
	tbl, err := objects.GetTable(s.db, sum)
	if err == objects.ErrKeyNotFound {
		return nil
	}
	if err != nil {
		return
	}
	for _, blk := range tbl.Blocks {
		if _, ok := s.commonBlocks[string(blk)]; !ok {
			s.objs.PushBack(object{Type: packfile.ObjectBlock, Sum: blk})
			s.commonBlocks[string(blk)] = struct{}{}
		}
	}
	s.buf.Reset()
	_, err = tbl.WriteTo(s.buf)
	if err != nil {
		return
	}
	b := make([]byte, s.buf.Len())
	copy(b, s.buf.Bytes())
	s.objs.PushBack(object{Type: packfile.ObjectTable, Content: b})

	return nil
}

func (s *ObjectSender) enqueueAllCommits() (err error) {
	for s.commits.Len() > 0 {
		com := s.commits.Remove(s.commits.Front()).(*objects.Commit)
		s.buf.Reset()
		_, err = com.WriteTo(s.buf)
		if err != nil {
			return
		}
		b := make([]byte, s.buf.Len())
		copy(b, s.buf.Bytes())
		s.objs.PushBack(object{Type: packfile.ObjectCommit, Content: b})
	}
	return nil
}

func (s *ObjectSender) WriteObjects(w io.Writer) (done bool, err error) {
	pw, err := packfile.NewPackfileWriter(w)
	if err != nil {
		return
	}
	var b []byte
	var size uint64
	var n int
	for s.objs.Len() > 0 {
		obj := s.objs.Remove(s.objs.Front()).(object)
		if obj.Content == nil {
			b, err = objects.GetBlockBytes(s.db, obj.Sum)
			if err != nil {
				return
			}
		} else {
			b = obj.Content
		}
		n, err = pw.WriteObject(obj.Type, b)
		if err != nil {
			return
		}
		size += uint64(n)
		if s.objs.Len() == 0 {
			if s.tables.Len() > 0 {
				if err = s.enqueueFrontTable(); err != nil {
					return
				}
			} else if s.commits.Len() > 0 {
				if err = s.enqueueAllCommits(); err != nil {
					return
				}
			}
		}
		if size >= s.maxPackfileSize {
			break
		}
	}
	return s.objs.Len() == 0 && s.commits.Len() == 0, nil
}
