package queue

import (
	"container/list"
	"sync"
	"sync/atomic"
	"time"
)

type Queue struct {
	messages  chan any
	list      list.List
	mutex     sync.RWMutex
	done      chan any
	listening atomic.Bool
}

func New(m <-chan any) *Queue {
	return &Queue{
		messages: m,
		done:     make(chan any, 1),
	}
}

func (q *Queue) Len() int {
	q.mutex.RLock()
	defer q.mutex.RUnlock()
	return q.list.Len()
}

func (q *Queue) WaitMessage(timeout time.Duration) (any, bool) {
	if q.Len() == 0 && len(q.messages) > 0 {
		return <-q.messages, true
	}

	ch := make(chan any, 1)
	q.mutex.Lock()
	elem := q.list.PushBack(ch)
	q.mutex.Unlock()

	q.pingListener()
	timer := time.NewTimer(time.Duration(t) * time.Second)

	select {
	case v := <-ch:
		timer.Stop()
		return v, true

	case <-timer.C:
		q.remove(elem)
		return nil, false

	case <-q.done:
		timer.Stop()
		q.remove(elem)
		return nil, false
	}
}

func (q *Queue) Close() {
	if q.Len() > 0 {
		close(q.done)
	}
}

func (q *Queue) pingListener() {
	if !q.listening.Load() {
		q.listening.Store(true)
		go q.listen()
	}
}

func (q *Queue) listen() {
	q.listening.Store(true)
	defer q.listening.Store(false)

	for q.Len() > 0 {
		select {
		case v := q.messages:
			q.mutex.Lock()
			elem := q.list.Front()
			q.list.Remove(elem)
			q.mutex.Unlock()
			ch := elem.Value.(chan any)
			ch <- v

		case <-q.done:
			return
		}
	}
}

func (q *Queue) remove(elem *list.Element) {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	q.list.Remove(elem)
}
