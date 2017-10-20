package schaloop

import (
	"errors"
	"runtime"
	"sync"
	"time"

	"github.com/xtuc/schaloop/monitoring"
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

func (work *work) safeExecute() {
	defer func() {
		if r := recover(); r != nil {
			go work.errorHandler(errors.New(r.(string)))
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
	queue   queue
	ticker  *time.Ticker
	monitor *monitoring.Monitor

	queueLock sync.Mutex
}

func NewEventLoop() *EventLoop {

	return &EventLoop{
		ticker:  time.NewTicker(LOOP_FREQ),
		monitor: monitoring.NewMonitor(),
	}
}

func (eventloop *EventLoop) QueueWork(name string, fn func()) {
	errorHandler := func(err error) {
		panic(err)
	}

	eventloop.QueueWorkWithError(name, fn, errorHandler)
}

func (eventloop *EventLoop) QueueWorkFromChannelWithError(name string, workChan chan interface{}, fn func(interface{}), errorHandler func(err error)) {
	wrappedFn := func() {

		go func() {
			for {
				data := <-workChan

				eventloop.QueueWorkWithError(name, func() {
					fn(data)
				}, errorHandler)

				runtime.Gosched()
			}
		}()
	}

	eventloop.QueueWorkWithError(name, wrappedFn, errorHandler)
}

func (eventloop *EventLoop) QueueWorkFromChannel(name string, workChan chan interface{}, fn func(interface{})) {
	errorHandler := func(err error) {
		panic(err)
	}

	eventloop.QueueWorkFromChannelWithError(name, workChan, fn, errorHandler)
}

func (eventloop *EventLoop) QueueWorkWithError(name string, fn func(), errorHandler func(err error)) {
	work := work{
		fn:           fn,
		name:         name,
		errorHandler: errorHandler,
	}

	eventloop.queueLock.Lock()
	eventloop.enqueue(work)
	eventloop.queueLock.Unlock()
}

func (eventloop *EventLoop) enqueue(work work) {
	eventloop.queue = append(queue{work}, eventloop.queue...)

	go eventloop.monitor.AddWork()
}

func (eventloop *EventLoop) dequeueWork() work {
	queue := eventloop.queue
	work := queue[len(queue)-1]

	eventloop.queue = queue[:len(queue)-1]

	go eventloop.monitor.SubWork()

	return work
}

func (eventloop *EventLoop) hasWork() bool {
	return len(eventloop.queue) > 0
}

func (eventloop *EventLoop) Stop() {
	eventloop.ticker.Stop()
}

func (eventloop *EventLoop) DumpMonitor() string {
	return eventloop.monitor.Dump()
}

func (eventloop *EventLoop) StartWithTimeout(timeout time.Duration) {

	go func() {
		for {
			var currentWork *work

			eventloop.queueLock.Lock()
			if eventloop.hasWork() {
				work := eventloop.dequeueWork()

				currentWork = &work
			}
			eventloop.queueLock.Unlock()

			// Block until next tick
			<-eventloop.ticker.C
			eventloop.monitor.Tick()

			if currentWork != nil {
				deadline := time.After(timeout)
				waitExecution := make(chan bool)

				eventloop.monitor.ExecutionStart()

				// Cancellable goroutine
				go func() {
					go func() {
						defer func() {
							currentWorkRef := currentWork

							if r := recover(); r != nil {
								waitExecution <- false
								go currentWorkRef.errorHandler(TIMEOUT_ERROR)
							}
						}()

						<-deadline

						if currentWork != nil {
							panic(TIMEOUT_ERROR)
						}
					}()

					currentWork.safeExecute()
					currentWork = nil
					waitExecution <- false
				}()

				<-waitExecution
				eventloop.monitor.ExecutionStop()
			}
		}

	}()
}
