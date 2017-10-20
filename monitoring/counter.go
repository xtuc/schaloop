package monitoring

import "sync/atomic"

type counter struct {
	count int64
}

func newCounter() *counter {
	return &counter{0}
}

func (counter *counter) Add(nbr int) {
	atomic.AddInt64(&counter.count, int64(nbr))
}

func (counter *counter) Sub(nbr int) {
	atomic.AddInt64(&counter.count, int64(nbr)*-1)
}

func (counter *counter) Get() int {
	return int(atomic.LoadInt64(&counter.count))
}
