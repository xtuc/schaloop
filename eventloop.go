package schaloop

import (
	"errors"
	"sync"
	"time"
)

var (
	TIMEOUT_ERROR = errors.New("Timeout")
	LOOP_FREQ     = time.Duration(100 * time.Microsecond)
)

type work struct {
	fn func()
	// Name for debugging purpose
	name         string
	errorHandler func(error)
}

func (work *work) safeExecute(timeout time.Duration) {
	defer func() {
		if r := recover(); r != nil {
			work.errorHandler(errors.New(r.(string)))
		}
	}()

	work.fn()
}

// Find a way to actually abort/interupt processing
// Panic to bailout?
func (work *work) abortWithError(err error) {
	work.errorHandler(err)
}

type queue []work

type EventLoop struct {
	queue  queue
	time   time.Time
	ticker *time.Ticker

	queueLock sync.Mutex
}

func NewEventLoop() *EventLoop {

	return &EventLoop{
		ticker: time.NewTicker(LOOP_FREQ),
	}
}

func (eventloop *EventLoop) QueueWork(name string, fn func()) {
	errorHandler := func(err error) {
		panic(err)
	}

	eventloop.QueueWorkWithError(name, fn, errorHandler)
}

func (eventloop *EventLoop) QueueWorkFromChannel(name string, workChan chan interface{}, fn func(interface{})) {
	errorHandler := func(err error) {
		panic(err)
	}

	wrappedFn := func() {

		go func() {
			for {
				data := <-workChan

				eventloop.QueueWorkWithError(name, func() {
					fn(data)
				}, errorHandler)
			}
		}()
	}

	eventloop.QueueWorkWithError(name, wrappedFn, errorHandler)
}

func (eventloop *EventLoop) QueueWorkWithError(name string, fn func(), errorHandler func(err error)) {
	work := work{
		fn:           fn,
		name:         name,
		errorHandler: errorHandler,
	}

	eventloop.enqueue(work)
}

func (eventloop *EventLoop) enqueue(work work) {
	eventloop.queueLock.Lock()
	eventloop.queue = append(queue{work}, eventloop.queue...)
	eventloop.queueLock.Unlock()
}

func (eventloop *EventLoop) dequeueWork() work {
	queue := eventloop.queue
	work := queue[len(queue)-1]

	eventloop.queue = queue[:len(queue)-1]

	return work
}

func (eventloop *EventLoop) hasWork() bool {
	return len(eventloop.queue) > 0
}

func (eventloop *EventLoop) Stop() {
	eventloop.ticker.Stop()
}

func (eventloop *EventLoop) StartWithTimeout(timeout time.Duration) {

	go func() {
		for {
			<-eventloop.ticker.C

			var currentWork *work

			eventloop.queueLock.Lock()
			if eventloop.hasWork() {
				work := eventloop.dequeueWork()

				currentWork = &work
			}
			eventloop.queueLock.Unlock()

			if currentWork != nil {

				eventloop.time = time.Now()
				currentWork.safeExecute(timeout)

				currentWork = nil
			}
		}

	}()
}
