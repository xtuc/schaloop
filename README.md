# schaloop

> Approach for event-driven Golang application.

## Design

It uses stack of work (that you enqueue, see API) which is processed by a single Goroutine. That way we can ensure the memory safety of the program.

### Experimentation: real thread

The Goroutine impose some technical restrictions:
- The work can not be resumed or aborted
- We don't control the scheduling (where and when the loop runs)

Go has a feature which allows any Goroutine to be assigned to a given system thread. Using the kernel primitives we are able to schedule and control the processing of the work more precisely (rlimits, priority, sleep, ...).

Unfortunately there also some limitations. It seems not possible to properly stop/kill the thread without halting the entire program.
