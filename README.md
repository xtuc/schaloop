# schaloop
--
    import "github.com/xtuc/schaloop"

Event loop - approach for event-driven Golang application.


### Design

It uses stack of work (that you enqueue, see API) which is processed by a single
Goroutine. All writes are linearized, that way we can ensure the memory safety
of the program.


Work timeout

To avoid stopping the loop for too long, schaloop provides a timeout mechanism.
We are able to interrupt work using a `panic`. The error won't be propagated.

Real thread (Not implemented)

The Goroutine impose some technical restrictions: - The work can not be resumed
or aborted - We don't control the scheduling (where and when the loop runs)

Go has a feature which allows any Goroutine to be assigned to a given system
thread. Using the kernel primitives we are able to schedule and control the
processing of the work more precisely (rlimits, priority, sleep, ...).

Unfortunately there also some limitations. It seems not possible to properly
stop/kill the thread without halting the entire program.

## Usage

```go
var (
	// Error will be returned if the work exceeded the timeout inside the event
	// loop.
	TIMEOUT_ERROR = errors.New("Timeout")

	// Frequency at which work will be processed inside of the event loop.
	LOOP_FREQ = time.Duration(100 * time.Microsecond)
)
```

#### type EventLoop

```go
type EventLoop struct {
}
```


#### func  NewEventLoop

```go
func NewEventLoop() *EventLoop
```
Initializes a new event loop Monitoring will be attached by default

#### func (*EventLoop) DumpMonitor

```go
func (eventloop *EventLoop) DumpMonitor() string
```
Dumps the monitoring inside the loop

#### func (EventLoop) DumpStack

```go
func (eventloop EventLoop) DumpStack() (str string)
```
Dumps the current execution stack of the loop

#### func (*EventLoop) QueueWork

```go
func (eventloop *EventLoop) QueueWork(name string, fn func())
```
Add new work to be processed eventually. Work resulting in a error will be
progragated.

#### func (*EventLoop) QueueWorkFromChannel

```go
func (eventloop *EventLoop) QueueWorkFromChannel(name string, work interface{}, fn func(interface{}))
```
Add new work each time a new message will be send in the channel. Work resulting
in a error will be progragated.

#### func (*EventLoop) QueueWorkFromChannelWithError

```go
func (eventloop *EventLoop) QueueWorkFromChannelWithError(name string, work interface{}, fn func(interface{}), errorHandler func(err error))
```
Same as QueueWorkFromChannel with a custom error handler

#### func (*EventLoop) QueueWorkWithError

```go
func (eventloop *EventLoop) QueueWorkWithError(name string, fn func(), errorHandler func(err error))
```
Same as QueueWork with a custom error handler

#### func (*EventLoop) StartWithTimeout

```go
func (eventloop *EventLoop) StartWithTimeout(timeout time.Duration)
```
Start the event loop

#### func (*EventLoop) Stop

```go
func (eventloop *EventLoop) Stop()
```
Stop the loop
