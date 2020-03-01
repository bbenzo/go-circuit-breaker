package go_circuit_breaker

import (
	"errors"
	"github.com/magiconair/properties/assert"
	"testing"
	"time"
)

func TestWhenThresholdExceededStateIsHalfOpenError(t *testing.T) {
	cb := NewCircuitBreaker(&Settings{threshold: 2, name: "test"})

	errFunc := func() (interface{}, error) {
		return nil, errors.New("i like to fail")
	}

	cb.Execute(errFunc)
	cb.Execute(errFunc)
	cb.Execute(errFunc)
	_, err := cb.Execute(errFunc)

	assert.Equal(t, err, errors.New("circuit half open. trying to recover"))
}

func TestWhenRecoverFailsStateIsOpen(t *testing.T) {
	cb := NewCircuitBreaker(&Settings{threshold: 2, name: "test", retryInterval: 1, retryMax: 5})

	errFunc := func() (interface{}, error) {
		return nil, errors.New("i like to fail")
	}

	cb.Execute(errFunc)
	cb.Execute(errFunc)
	cb.Execute(errFunc)
	_, err := cb.Execute(errFunc)

	assert.Equal(t, err, errors.New("circuit half open. trying to recover"))
	assert.Equal(t, cb.GetState(), HalfOpen)

	// sleep until attempt to recover is over
	time.Sleep(time.Second * 7)

	assert.Equal(t, cb.GetState(), Open)

	// fail immediately with alert
	_, err = cb.Execute(errFunc)
	assert.Equal(t, err, errors.New("test circuit breaker open"))
}

func TestWhenRecoverSucceedsStateIsClosed(t *testing.T) {
	cb := NewCircuitBreaker(&Settings{threshold: 2, name: "test", retryInterval: 1, retryMax: 5})

	then := time.Now().Add(time.Second * 3)
	testFunc := func() (interface{}, error) {
		now := time.Now()
		if now.Unix() > then.Unix() {
			return "yay", nil
		}
		return nil, errors.New("i like to fail")
	}

	// execute with error response until state is half open
	cb.Execute(testFunc)
	cb.Execute(testFunc)
	cb.Execute(testFunc)
	_, err := cb.Execute(testFunc)

	assert.Equal(t, err, errors.New("circuit half open. trying to recover"))
	assert.Equal(t, cb.GetState(), HalfOpen)

	// sleep until attempt to recover is over
	time.Sleep(time.Second * 5)

	// state is closed and new retry resolves in response
	assert.Equal(t, cb.GetState(), Closed)

	res, err := cb.Execute(testFunc)
	assert.Equal(t, err, nil)
	assert.Equal(t, res, "yay")
}
