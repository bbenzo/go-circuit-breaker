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

// Strategy holds variables to configure circuit breaker
type Strategy struct {
	Threshold     int
	RetryInterval int
	RetryMax      int
}

type circuitBreaker struct {
	name              string
	strategy          *Strategy
	state             State
	consecutiveErrors int
}

// CircuitBreaker defines the circuit breaker decorator interface
type CircuitBreaker interface {
	Execute(func() (interface{}, error)) (interface{}, error)
	GetState() State
	GetName() string
}

// GetName returns name of circuit breaker
func (c *circuitBreaker) GetName() string {
	return c.name
}

// GetState returns state of circuit breaker
func (c *circuitBreaker) GetState() State {
	return c.state
}

// NewCircuitBreaker returns new instance of circuit breaker
func NewCircuitBreaker(name string, strategy *Strategy) CircuitBreaker {
	if strategy.Threshold <= 0 {
		strategy.Threshold = defaultErrorThreshold
	}

	if strategy.RetryMax <= 0 {
		strategy.RetryMax = defaultRetryMax
	}

	if strategy.RetryInterval <= 0 {
		strategy.RetryInterval = defaultRetryInterval
	}

	return &circuitBreaker{
		name:              name,
		strategy:          strategy,
		state:             Closed,
		consecutiveErrors: 0,
	}
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
		message := fmt.Sprintf("%v circuit breaker open", c.name)
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
	if c.consecutiveErrors > c.strategy.Threshold {
		c.state = HalfOpen
		go c.recover(f)
	}
}

func (c *circuitBreaker) recover(f func() (interface{}, error)) {
	retries := 0
	for c.state == HalfOpen {
		// Open circuit breaker when recovering fails
		if retries > c.strategy.RetryMax {
			c.state = Open
			return
		}

		time.Sleep(time.Second * time.Duration(c.strategy.RetryInterval))

		// set state to closed if request is successful
		_, err := f()
		if err == nil {
			c.state = Closed
		}

		retries++
	}
}
