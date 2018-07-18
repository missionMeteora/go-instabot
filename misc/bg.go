package misc

import "sync"

type Backgrounder struct {
	wg sync.WaitGroup
}

func (bg *Backgrounder) Add(fn func()) {
	bg.wg.Add(1)
	go func() {
		defer bg.wg.Done() // panic protection
		fn()
	}()
}

func (bg *Backgrounder) Wait() { bg.wg.Wait() }
