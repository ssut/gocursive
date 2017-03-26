package main

import "github.com/sethgrid/multibar"

type ProgressPool struct {
	root         *multibar.BarContainer
	progressBars []*ProgressBar
}

type ProgressBar struct {
	barFunc multibar.ProgressFunc
	using   bool
}

func NewProgrssPool(size int) *ProgressPool {
	root, _ := multibar.New()

	bars := make([]*ProgressBar, size)
	for i := range bars {
		pbar := &ProgressBar{
			barFunc: root.MakeBar(1, ""),
			using:   false,
		}
		bars[i] = pbar
	}

	pool := &ProgressPool{
		root:         root,
		progressBars: bars,
	}

	return pool
}

func (pool *ProgressPool) Acquire() multibar.ProgressFunc {
	for _, bar := range pool.progressBars {
		if !bar.using {
			bar.using = true
			return bar.barFunc
		}
	}

	return nil
}

func (pool *ProgressPool) Release(barFunc multibar.ProgressFunc) {
}
