/*
Event loop - approach for event-driven Golang application.

Design

It uses stack of work (that you enqueue, see API) which is processed by a single Goroutine. All writes are linearized, that way we can ensure the memory safety of the program.

Work timeout

To avoid stopping the loop for too long, schaloop provides a timeout mechanism. We are able to interrupt work using a `panic`. The error won't be propagated.

Real thread (Not implemented)

The Goroutine impose some technical restrictions:
- The work can not be resumed or aborted
- We don't control the scheduling (where and when the loop runs)

Go has a feature which allows any Goroutine to be assigned to a given system thread. Using the kernel primitives we are able to schedule and control the processing of the work more precisely (rlimits, priority, sleep, ...).

Unfortunately there also some limitations. It seems not possible to properly stop/kill the thread without halting the entire program.
*/
package schaloop

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/xtuc/schaloop/monitoring"
)

var (
	// Error will be returned if the work exceeded the timeout inside the event
	// loop.
	TIMEOUT_ERROR = errors.New("Timeout")

	// Frequency at which work will be processed inside of the event loop.
	LOOP_FREQ = time.Duration(100 * time.Microsecond)
)

type work struct {
	fn func()
	// Name for debugging purpose
	name         string
	errorHandler func(error)
}

func (work *work) safeExecute() {
	defer func() {
		if err := recover(); err != nil {

			if errString, isString := err.(string); isString {
				work.errorHandler(errors.New(errString))
			} else {
				work.errorHandler(fmt.Errorf("An error occurred: %s", err))
			}
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

// Initializes a new event loop
// Monitoring will be attached by default
func NewEventLoop() *EventLoop {

	return &EventLoop{
		ticker:  time.NewTicker(LOOP_FREQ),
		monitor: monitoring.NewMonitor(),
	}
}

// Add new work to be processed eventually.
// Work resulting in a error will be progragated.
func (eventloop *EventLoop) QueueWork(name string, fn func()) {
	errorHandler := func(err error) {
		panic(err)
	}

	eventloop.QueueWorkWithError(name, fn, errorHandler)
}

// Same as QueueWorkFromChannel with a custom error handler
func (eventloop *EventLoop) QueueWorkFromChannelWithError(name string, work interface{}, fn func(interface{}), errorHandler func(err error)) {
	workChan, ok := work.(chan interface{})
	if !ok {
		panic("Work is not a chan")
	}

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

// Add new work each time a new message will be send in the channel.
// Work resulting in a error will be progragated.
func (eventloop *EventLoop) QueueWorkFromChannel(name string, work interface{}, fn func(interface{})) {
	workChan, ok := work.(chan interface{})
	if !ok {
		panic("Work is not a chan")
	}

	errorHandler := func(err error) {
		panic(err)
	}

	eventloop.QueueWorkFromChannelWithError(name, workChan, fn, errorHandler)
}

// Same as QueueWork with a custom error handler
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

// Stop the loop
func (eventloop *EventLoop) Stop() {
	eventloop.ticker.Stop()
}

// Dumps the monitoring inside the loop
func (eventloop *EventLoop) DumpMonitor() string {
	return eventloop.monitor.Dump()
}

// Dumps the current execution stack of  the loop
func (eventloop EventLoop) DumpStack() (str string) {
	for k, work := range eventloop.queue {
		str += fmt.Sprintf("[%d] fn: %p work: %p name: %s\n\r", k, eventloop.queue[k].fn, &eventloop.queue[k], work.name)
	}

	return str
}

// Start the event loop
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
