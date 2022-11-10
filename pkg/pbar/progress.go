package pbar

import (
	"io"

	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

const (
	UnitKiB int = decor.UnitKiB
	UnitKB  int = decor.UnitKB
)

type Container interface {
	NewBar(total int64, name string, unit int) Bar
	Wait()
}

type noopContainer struct{}

func (p *noopContainer) NewBar(total int64, name string, unit int) Bar {
	return &noopBar{}
}

func (p *noopContainer) Wait() {}

func NewNoopContainer() Container {
	return &noopContainer{}
}

type container struct {
	p *mpb.Progress
}

func NewContainer(out io.Writer) Container {
	p := &container{
		p: mpb.New(mpb.WithOutput(out)),
	}
	return p
}

func (p *container) NewBar(total int64, name string, unit int) Bar {
	options := []mpb.BarOption{
		mpb.PrependDecorators(decor.Name(name, decor.WC{W: len(name) + 1, C: decor.DidentRight}), decor.Counters(unit, "%d / %d")),
		mpb.BarRemoveOnComplete(),
	}
	if total > 0 {
		options = append(options,
			mpb.AppendDecorators(decor.Percentage(decor.WC{W: 5, C: decor.DidentRight}), decor.Elapsed(decor.ET_STYLE_GO)),
		)
	} else {
		options = append(options,
			mpb.AppendDecorators(decor.Elapsed(decor.ET_STYLE_GO)),
		)
	}
	var b = p.p.New(total,
		mpb.BarStyle().Lbound("[").Filler("=").Tip(">").Padding(" ").Rbound("]"),
		options...,
	)
	b.EnableTriggerComplete()
	return &bar{
		b:     b,
		total: total,
	}
}

func (p *container) Wait() {
	p.p.Wait()
}

// NewProgressBar not only combines NewContainer and NewBar but also
// ensure that when bar.Done is called, container.Wait is also called.
func NewProgressBar(out io.Writer, total int64, name string, unit int) Bar {
	c := NewContainer(out)
	_c := c.(*container)
	b := c.NewBar(total, name, unit)
	_b := b.(*bar)
	_b.p = _c.p
	return _b
}
