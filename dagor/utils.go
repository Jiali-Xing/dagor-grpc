package dagor

import (
	"fmt"
	"sync/atomic"
	"time"
)

// logger is to mock a sophisticated logging system. To simplify the example, we just print out the content.
func logger(format string, a ...interface{}) {
	if debug {
		// print to stdout with timestamp
		timestamp := time.Now().Format("2006-01-02T15:04:05.999999999-07:00")
		fmt.Printf("LOG: "+timestamp+"|\t"+format+"\n", a...)
	}
}

func (d *Dagor) ReadN() int64 {
	return atomic.LoadInt64(&d.N)
}

func (d *Dagor) UpdateN(newN int64) {
	atomic.StoreInt64(&d.N, newN)
}

func (d *Dagor) IncrementN() {
	atomic.AddInt64(&d.N, 1)
}

func (d *Dagor) DecrementN() {
	atomic.AddInt64(&d.N, -1)
}
