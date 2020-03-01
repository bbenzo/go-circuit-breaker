package go_circuit_breaker

import (
	"errors"
	"fmt"
	"time"
)

type State int

const (
	Closed   State = 1
	HalfOpen State = 2
	Open     State = 3
)

const defaultErrorThreshold = 5
const defaultRetryInterval = 5
const defaultRetryMax = 5

// Settings holds variables to configure circuit breaker
type Settings struct {
	name          string
	threshold     int
	retryInterval int
	retryMax      int
}

type circuitBreaker struct {
	settings          *Settings
	state             State
	consecutiveErrors int
}

func (c *circuitBreaker) GetName() string {
	return c.settings.name
}

// CircuitBreaker defines the circuit breaker decorator interface
type CircuitBreaker interface {
	Execute(func() (interface{}, error)) (interface{}, error)
	GetState() State
	GetName() string
}

// NewCircuitBreaker returns new instance of circuit breaker
func NewCircuitBreaker(settings *Settings) CircuitBreaker {
	if settings.threshold <= 0 {
		settings.threshold = defaultErrorThreshold
	}

	if settings.retryMax <= 0 {
		settings.retryMax = defaultRetryMax
	}

	if settings.retryInterval <= 0 {
		settings.retryInterval = defaultRetryInterval
	}

	return &circuitBreaker{
		settings:          settings,
		state:             Closed,
		consecutiveErrors: 0,
	}
}

// GetState returns state of circuit breaker
func (c *circuitBreaker) GetState() State {
	return c.state
}

// Execute executes a function wrapped in a circuit breaker pattern
func (c *circuitBreaker) Execute(f func() (interface{}, error)) (interface{}, error) {
	switch c.state {
	case Closed:
		res, err := f()
		if err != nil {
			c.handleError(f)
			return res, err
		}

		c.handleSuccess()
	case HalfOpen:
		return nil, errors.New("circuit half open. trying to recover")
	case Open:
		message := fmt.Sprintf("%v circuit breaker open", c.settings.name)
		fmt.Printf("ALERT: %v", message)
		return nil, errors.New(message)
	}
	return f()
}

func (c *circuitBreaker) handleSuccess() {
	c.consecutiveErrors = 0
}

func (c *circuitBreaker) handleError(f func() (interface{}, error)) {
	c.consecutiveErrors++
	if c.consecutiveErrors > c.settings.threshold {
		c.state = HalfOpen
		go c.recover(f)
	}
}

// recover executes function every 5 seconds to check if it still returns an error
func (c *circuitBreaker) recover(f func() (interface{}, error)) {
	retries := 0
	for c.state == HalfOpen {
		// Open circuit breaker when recovering fails
		if retries > c.settings.retryMax {
			c.state = Open
			return
		}

		time.Sleep(time.Second * time.Duration(c.settings.retryInterval))

		// set state to closed if request is successful
		_, err := f()
		if err == nil {
			c.state = Closed
		}

		retries++
	}
}
