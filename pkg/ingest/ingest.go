package ingest

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"

	"github.com/mmcloughlin/meow"
	"github.com/wrgl/core/pkg/objects"
	"github.com/wrgl/core/pkg/sorter"
)

type ProgressBar interface {
	Add(int) error
	Finish() error
}

func insertBlock(db objects.Store, pt ProgressBar, tbl *objects.Table, tblIdx [][]string, blocks <-chan *sorter.Block, wg *sync.WaitGroup, errChan chan<- error) {
	buf := bytes.NewBuffer(nil)
	defer wg.Done()
	dec := objects.NewStrListDecoder(true)
	hash := meow.New(0)
	e := objects.NewStrListEditor(tbl.PK)
	for blk := range blocks {
		// write block and add block to table
		sum, err := objects.SaveBlock(db, blk.Block)
		if err != nil {
			errChan <- err
			return
		}
		tbl.Blocks[blk.Offset] = sum

		// write block index and add pk sums to table index
		idx, err := objects.IndexBlockFromBytes(dec, hash, e, blk.Block, tbl.PK)
		if err != nil {
			errChan <- err
			return
		}
		buf.Reset()
		idx.WriteTo(buf)
		err = objects.SaveBlockIndex(db, sum, buf.Bytes())
		if err != nil {
			errChan <- err
			return
		}
		tblIdx[blk.Offset] = blk.PK
		if pt != nil {
			pt.Add(1)
		}
	}
}

func IngestTableFromBlocks(db objects.Store, columns []string, pk []uint32, rowsCount uint32, blocks <-chan *sorter.Block, numWorkers int, pt ProgressBar) ([]byte, error) {
	tbl := objects.NewTable(columns, pk, rowsCount)
	tblIdx := make([][]string, objects.BlocksCount(rowsCount))
	wg := &sync.WaitGroup{}
	errChan := make(chan error, numWorkers)
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go insertBlock(db, pt, tbl, tblIdx, blocks, wg, errChan)
	}
	wg.Wait()
	close(errChan)
	err, ok := <-errChan
	if ok {
		return nil, err
	}
	if pt != nil {
		if err := pt.Finish(); err != nil {
			return nil, err
		}
	}

	// write and save table
	buf := bytes.NewBuffer(nil)
	_, err = tbl.WriteTo(buf)
	if err != nil {
		return nil, err
	}
	sum, err := objects.SaveTable(db, buf.Bytes())
	if err != nil {
		return nil, err
	}

	// write and save table index
	buf.Reset()
	enc := objects.NewStrListEncoder(true)
	_, err = objects.WriteBlockTo(enc, buf, tblIdx)
	if err != nil {
		return nil, err
	}
	err = objects.SaveTableIndex(db, sum, buf.Bytes())
	if err != nil {
		return nil, err
	}

	return sum, nil
}

func IngestTable(db objects.Store, f io.ReadCloser, pk []string, sortRunSize uint64, numWorkers int, sortPT, blkPT ProgressBar) ([]byte, error) {
	s, err := sorter.NewSorter(sortRunSize, sortPT)
	if err != nil {
		return nil, err
	}
	defer s.Close()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		if err := s.Close(); err != nil {
			fmt.Fprint(os.Stderr, err.Error())
			os.Exit(1)
		}
		os.Exit(0)
	}()
	numWorkers -= 2
	if numWorkers <= 0 {
		numWorkers = 1
	}
	err = s.SortFile(f, pk)
	if err != nil {
		return nil, err
	}
	errChan := make(chan error, numWorkers)
	blocks := s.SortedBlocks(nil, errChan)
	sum, err := IngestTableFromBlocks(db, s.Columns, s.PK, s.RowsCount, blocks, numWorkers, blkPT)
	if err != nil {
		return nil, err
	}
	close(errChan)
	err, ok := <-errChan
	if ok {
		return nil, err
	}
	return sum, nil
}
