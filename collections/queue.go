package collections

import "sync"

type Queue struct {
	start  int
	end    int
	buf    []interface{}
	cond   *sync.Cond
	closed bool
}

func (q *Queue) bufferLen() int {
	return (q.end + cap(q.buf) - q.start) % cap(q.buf)
}

func (q *Queue) Len() int {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	return q.bufferLen()
}

func (q *Queue) Close() bool {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()

	if q.closed {
		return false
	}

	q.closed = true
	q.cond.Broadcast()
	return true
}

func (q *Queue) Put(data interface{}) bool {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()

	if q.closed {
		return false
	}

	// if there is only 1 free slot, allocate more
	var old_cap = cap(q.buf)
	if (q.end+1)%old_cap == q.start {
		buf := make([]interface{}, cap(q.buf)*2)
		if q.end > q.start {
			copy(buf, q.buf[q.start:q.end])
		} else if q.end < q.start {
			copy(buf, q.buf[q.start:old_cap])
			copy(buf[old_cap-q.start:], q.buf[0:q.end])
		}
		q.buf = buf
		q.start = 0
		q.end = old_cap - 1
	}

	q.buf[q.end] = data
	q.end = (q.end + 1) % cap(q.buf)
	q.cond.Signal()
	return true
}

func (q *Queue) Pop() (interface{}, bool) {
	for {
		q.cond.L.Lock()
		if q.bufferLen() > 0 {
			data := q.buf[q.start]
			q.start = (q.start + 1) % cap(q.buf)
			q.cond.L.Unlock()
			return data, true
		}
		if q.closed {
			q.cond.L.Unlock()
			return nil, false
		}
		q.cond.Wait()
		q.cond.L.Unlock()
	}
}

func NewQueue(sz int) *Queue {
	var l sync.Mutex
	return &Queue{
		buf:   make([]interface{}, sz),
		start: 0,
		end:   0,
		cond:  sync.NewCond(&l),
	}
}
