package monitoring

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAddAndSubWork(t *testing.T) {
	m := NewMonitor()

	assert.Equal(t, m.lag.Get(), 0)

	m.AddWork()
	m.AddWork()

	assert.Equal(t, m.lag.Get(), 2)
	assert.Equal(t, m.ticks.Get(), 0)

	m.SubWork()
	m.SubWork()

	assert.Equal(t, m.lag.Get(), 0)
}

func TestTick(t *testing.T) {
	m := NewMonitor()

	assert.Equal(t, m.ticks.Get(), 0)

	m.Tick()
	m.Tick()

	assert.Equal(t, m.ticks.Get(), 2)
	assert.Equal(t, m.lag.Get(), 0)
}

func TestExecution(t *testing.T) {
	delay := 10 * time.Millisecond
	m := NewMonitor()

	assert.Equal(t, m.execution.Get(), 0)

	m.ExecutionStart()
	<-time.After(delay)
	m.ExecutionStop()

	assert.NotEqual(t, m.execution.Get(), 0)
}
