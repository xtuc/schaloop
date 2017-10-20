package monitoring

import (
	"fmt"
	"time"
)

type Monitor struct {
	currentExecutionStart *time.Time

	ticks     *counter
	lag       *counter
	execution *counter
}

func NewMonitor() *Monitor {
	return &Monitor{
		ticks:     newCounter(),
		lag:       newCounter(),
		execution: newCounter(),
	}
}

func (m *Monitor) AddWork() {
	m.lag.Add(1)
}

func (m *Monitor) SubWork() {
	m.lag.Sub(1)
}

func (m *Monitor) Tick() {
	m.ticks.Add(1)
}

func (m *Monitor) ExecutionStart() {
	now := time.Now()
	m.currentExecutionStart = &now
}

func (m *Monitor) ExecutionStop() {
	if m.currentExecutionStart == nil {
		return
	}

	t := time.Now()
	duration := t.Sub(*m.currentExecutionStart)

	m.execution.Add(int(duration.Nanoseconds()))
}

func (m *Monitor) Dump() string {
	return fmt.Sprintf(
		"ticks n: %d, lag: %d, execution (ns): %d",
		m.ticks.Get(),
		m.lag.Get(),
		m.execution.Get(),
	)
}
