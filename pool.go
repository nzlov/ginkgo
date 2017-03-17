package ginkgo

import "sync"

type pool struct {
	sync.Mutex
	pool  []*conn
	index int
}

func newPool(n int) *pool {
	return &pool{
		index: 0,
		pool:  make([]*conn, n, n),
	}
}
func (p *pool) Get() *conn {
	p.Lock()
	defer p.Unlock()
	if p.index > 0 {
		c := p.pool[p.index-1]
		p.pool[p.index-1] = nil
		p.index--
		return c
	} else {
		return nil
	}
}

func (p *pool) Put(c *conn) {
	p.Lock()
	defer p.Unlock()
	if p.index+1 > cap(p.pool) {
		return
	}
	p.pool[p.index] = c
	p.index++
}
