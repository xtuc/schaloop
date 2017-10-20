package schaloop

import (
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEventLoopEnsureOrder(t *testing.T) {
	e := NewEventLoop()
	e.StartWithTimeout(time.Duration(1 * time.Second))

	order := make([]int, 0)

	waitChan := make(chan bool)

	e.QueueWork("1", func() {
		order = append(order, 1)
	})

	e.QueueWork("2", func() {
		order = append(order, 2)
		waitChan <- false
	})

	<-waitChan
	e.Stop()

	assert.Equal(t, order, []int{1, 2})
}

func TestEventLoopHandleError(t *testing.T) {
	e := NewEventLoop()
	e.StartWithTimeout(time.Duration(1 * time.Second))

	calledErrorHandler := false

	waitChan := make(chan bool)

	e.QueueWorkWithError("1", func() {
		panic("foo")
	}, func(err error) {
		assert.Equal(t, err.Error(), "foo")

		calledErrorHandler = true
	})

	e.QueueWork("shutdown", func() {
		waitChan <- false
	})

	<-waitChan
	e.Stop()

	assert.True(t, calledErrorHandler)
}

func TestEventRegisterChannel(t *testing.T) {
	e := NewEventLoop()
	e.StartWithTimeout(time.Duration(1 * time.Second))

	waitChan := make(chan bool)
	workChan := make(chan interface{})

	e.QueueWorkFromChannel("test-workChan", workChan, func(data interface{}) {
		if data == nil {
			waitChan <- false
		}
	})

	workChan <- 1
	workChan <- 2
	workChan <- nil

	<-waitChan
	e.Stop()
}

func TestEventLoopWriteConsistency(t *testing.T) {
	rand.Seed(time.Now().Unix())

	e := NewEventLoop()
	e.StartWithTimeout(time.Duration(5 * time.Second))

	waitChan := make(chan bool)

	iterations := 10000
	count := 0

	for i := 0; i < iterations; i++ {
		name := "test-" + strconv.Itoa(i)

		go e.QueueWork(name, func() {
			randomDelay := time.Duration(rand.Intn(10-1) + 1)
			<-time.After(randomDelay * time.Microsecond)
			count++

			if count == iterations {
				waitChan <- false
			}
		})
	}

	go func() {
		<-time.After(15 * time.Second)
		waitChan <- false
	}()

	<-waitChan
	e.Stop()

	assert.Equal(t, count, iterations)
}
