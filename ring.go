package main

import (
	"bytes"
	"sync"
)

type CircularBuffer struct {
	sync.Mutex
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

func (c *CircularBuffer) Write(p []byte) (n int, err error) {
	c.Lock()
	defer c.Unlock()

	if c.size == c.capacity {
		c.head = (c.head + 1) % c.capacity
	} else {
		c.size++
	}

	return c.buffer[c.head].Write(p)
}

func (c *CircularBuffer) GetAll() [][]byte {
	c.Lock()
	defer c.Unlock()

	var result [][]byte
	for i := 0; i < c.size; i++ {
		result = append(result, c.buffer[(c.head+i)%c.capacity].Bytes())
	}
	return result
}
