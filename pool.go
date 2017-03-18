package ginkgo

import "sync"

type pool struct {
	sync.Mutex
	pool      []*conn
	index     int
	num       int
	limit     int
	removemap map[*conn]bool
}

func newPool(n int) *pool {
	return &pool{
		index:     0,
		num:       0,
		limit:     n,
		pool:      make([]*conn, n, n),
		removemap: make(map[*conn]bool),
	}
}
func (p *pool) Get() *conn {
	p.Lock()
	if p.index > 0 {
		c := p.pool[p.index-1]
		p.pool[p.index-1] = nil
		p.index--
		p.Unlock()
		return c
	} else {
		p.Unlock()
		return nil
	}
}
func (p *pool) Add(c *conn) bool {
	p.Lock()
	if p.num >= cap(p.pool) {
		p.Unlock()
		return false
	}
	p.pool[p.index] = c
	p.index++
	p.num++
	p.Unlock()
	return true
}

func (p *pool) Put(c *conn) {
	p.Lock()
	if p.index+1 > cap(p.pool) {
		p.Unlock()
		return
	}
	if ok := p.removemap[c]; ok {
		p.num--
		delete(p.removemap, c)
		p.Unlock()
		//glog.NewTagField("Pool").Set("removemap", p.removemap).Debugln("Put Remove")
		return
	}
	p.pool[p.index] = c
	p.index++
	p.Unlock()
}

func (p *pool) Remove(c *conn) {
	p.Lock()
	index := -1
	for i, v := range p.pool {
		if c == v {
			index = i
			break
		}
	}
	if index > -1 {
		for i := index; i < p.index-1; i++ {
			p.pool[i] = p.pool[i+1]
		}
		p.pool[p.index-1] = nil
		p.index--
		p.num--
		p.Unlock()
		//glog.NewTagField("Pool").Set("index", p.index).Set("num", p.num).Debugln("Remove")
		return
	}
	p.removemap[c] = true
	p.Unlock()
}
