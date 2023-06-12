package main

import (
	"bytes"
	"sync"
)

type CircularBuffer struct {
	mu       sync.Mutex
	buffer   []bytes.Buffer
	capacity int
	head     int
	size     int
}

func NewCircularBuffer(capacity int) *CircularBuffer {
	return &CircularBuffer{
		buffer:   make([]bytes.Buffer, capacity),
		capacity: capacity,
	}
}

func (c *CircularBuffer) Write(p []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.size < c.capacity {
		c.size++
	}

	c.buffer[c.head].Reset()
	c.buffer[c.head].Write(p)

	c.head = (c.head + 1) % c.capacity
}

func (c *CircularBuffer) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.head = 0
	c.size = 0
}

func (c *CircularBuffer) GetAll() [][]byte {
	c.mu.Lock()
	defer c.mu.Unlock()

	result := make([][]byte, c.size)

	if c.size < c.capacity {
		for i := 0; i < c.size; i++ {
			result[i] = c.buffer[i].Bytes()
		}
		return result
	} else {
		for i := 0; i < c.capacity; i++ {
			result[i] = c.buffer[(i+c.head)%c.capacity].Bytes()
		}
	}
	return result
}
