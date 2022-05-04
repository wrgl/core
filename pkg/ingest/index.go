// SPDX-License-Identifier: Apache-2.0
// Copyright © 2022 Wrangle Ltd

package ingest

import (
	"bytes"
	"fmt"
	"log"

	"github.com/pckhoi/meow"
	"github.com/wrgl/wrgl/pkg/objects"
	"github.com/wrgl/wrgl/pkg/slice"
)

func IndexTable(db objects.Store, tblSum []byte, tbl *objects.Table, logger *log.Logger) error {
	var (
		tblIdx    = make([][]string, len(tbl.Blocks))
		buf       = bytes.NewBuffer(nil)
		enc       = objects.NewStrListEncoder(true)
		hash      = meow.New(0)
		blk       [][]string
		err       error
		bb        []byte
		blkIdxSum []byte
	)
	if logger != nil {
		logger.Printf("Indexing table %x\n", tblSum)
	}
	var idxSum = make([]byte, meow.Size)
	for i, sum := range tbl.Blocks {
		blk, bb, err = objects.GetBlock(db, bb, sum)
		if err != nil {
			return fmt.Errorf("GetBlock: %v", err)
		}
		if len(tbl.PK) > 0 {
			tblIdx[i] = slice.IndicesToValues(blk[0], tbl.PK)
		} else {
			tblIdx[i] = make([]string, len(blk[0]))
			copy(tblIdx[i], blk[0])
		}
		idx, err := objects.IndexBlock(enc, hash, blk, tbl.PK)
		if err != nil {
			return fmt.Errorf("IndexBlock: %v", err)
		}
		buf.Reset()
		_, err = idx.WriteTo(buf)
		if err != nil {
			return fmt.Errorf("idx.WriteTo: %v", err)
		}
		if logger != nil {
			hash.Reset()
			hash.Write(buf.Bytes())
			hash.SumTo(idxSum)
			logger.Printf("  block %x (indexSum %x)\n", sum, idxSum)
		}
		blkIdxSum, bb, err = objects.SaveBlockIndex(db, bb, buf.Bytes())
		if err != nil {
			return fmt.Errorf("objects.SaveBlockIndex: %v", err)
		}
		if !bytes.Equal(blkIdxSum, tbl.BlockIndices[i]) {
			return fmt.Errorf("block index at offset %d has different sum: %x != %x", i, blkIdxSum, tbl.BlockIndices[i])
		}
	}
	buf.Reset()
	_, err = objects.WriteBlockTo(enc, buf, tblIdx)
	if err != nil {
		return fmt.Errorf("objects.WriteBlockTo: %v", err)
	}
	return objects.SaveTableIndex(db, tblSum, buf.Bytes())
}
