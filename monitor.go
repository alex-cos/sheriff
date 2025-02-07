package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/alex-cos/sheriff/nodes"
)

// Monitor periodically checks the status of all managed services and
// optionally restarts any that have stopped.
type Monitor struct {
	logger  *slog.Logger
	root    *nodes.ServiceNode
	period  time.Duration
	restart bool
	stop    chan bool
	done    chan bool
}

// NewMonitor creates a Monitor that checks the service tree at the given
// interval and conditionally restarts stopped services.
func NewMonitor(
	logger *slog.Logger,
	root *nodes.ServiceNode,
	period time.Duration,
	restart bool,
) *Monitor {
	return &Monitor{
		logger:  logger,
		root:    root,
		period:  period,
		restart: restart,
	}
}

// Start launches the monitor loop in a background goroutine.
// It logs status at every tick and restarts services if restart is enabled.
func (m *Monitor) Start() {
	m.stop = make(chan bool)
	m.done = make(chan bool)

	go func() {
		ticker := time.NewTicker(m.period)
		defer func() {
			ticker.Stop()
			m.logger.Info("monitor is stopping")
			m.done <- true
		}()

		m.logger.Info("monitor is starting")
		for {
			select {
			case <-ticker.C:
				fmt.Fprintf(os.Stdout, "status: %v\n", m.root)
				m.logger.Info("status", slog.String("root", m.Status()))
				if m.restart {
					m.root.RestartStopped(context.Background())
				}
			case <-m.stop:
				return
			}
		}
	}()
}

// Stop signals the monitor goroutine to exit and waits for it to finish.
func (m *Monitor) Stop() {
	if m.stop != nil {
		m.stop <- true
		<-m.done
	}
}

// Status returns a human-readable summary of the service tree
// ("All running", "Partially running", or "Not running").
func (m *Monitor) Status() string {
	total, running := m.root.Status()

	switch {
	case running == 0:
		return "Not running"
	case total == running:
		return "All running"
	case total != running:
		return "Partially running"
	default:
		return "Unknown"
	}
}
