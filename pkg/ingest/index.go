package ingest

import (
	"bytes"
	"fmt"
	"io"

	"github.com/mmcloughlin/meow"
	"github.com/wrgl/wrgl/pkg/objects"
	"github.com/wrgl/wrgl/pkg/slice"
)

func IndexTable(db objects.Store, tblSum []byte, tbl *objects.Table, debugOut io.Writer) error {
	tblIdx := make([][]string, len(tbl.Blocks))
	buf := bytes.NewBuffer(nil)
	enc := objects.NewStrListEncoder(true)
	hash := meow.New(0)
	if debugOut != nil {
		fmt.Fprintf(debugOut, "Indexing table %x\n", tblSum)
	}
	for i, sum := range tbl.Blocks {
		blk, err := objects.GetBlock(db, sum)
		if err != nil {
			return err
		}
		tblIdx[i] = slice.IndicesToValues(blk[0], tbl.PK)
		idx, err := objects.IndexBlock(enc, hash, blk, tbl.PK)
		if err != nil {
			return err
		}
		_, err = idx.WriteTo(buf)
		if err != nil {
			return err
		}
		if debugOut != nil {
			hash.Reset()
			hash.Write(buf.Bytes())
			idxSum := hash.Sum(nil)
			fmt.Fprintf(debugOut, "  block %x (indexSum %x)\n", sum, idxSum)
		}
		tblIdxSum, err := objects.SaveBlockIndex(db, buf.Bytes())
		if err != nil {
			return err
		}
		if !bytes.Equal(tblIdxSum, tbl.BlockIndices[i]) {
			return fmt.Errorf("block index at offset %d has different sum: %x != %x", i, tblIdxSum, tbl.BlockIndices[i])
		}
		buf.Reset()
	}
	_, err := objects.WriteBlockTo(enc, buf, tblIdx)
	if err != nil {
		return err
	}
	return objects.SaveTableIndex(db, tblSum, buf.Bytes())
}
