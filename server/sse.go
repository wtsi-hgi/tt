/*******************************************************************************
 * Copyright (c) 2025 Genome Research Ltd.
 *
 * Author: Sendu Bala <sb10@sanger.ac.uk>
 * This code largely taken from babyapi by CalvinMclean,
 * Apache License, Version 2.0
 *
 * Permission is hereby granted, free of charge, to any person obtaining
 * a copy of this software and associated documentation files (the
 * "Software"), to deal in the Software without restriction, including
 * without limitation the rights to use, copy, modify, merge, publish,
 * distribute, sublicense, and/or sell copies of the Software, and to
 * permit persons to whom the Software is furnished to do so, subject to
 * the following conditions:
 *
 * The above copyright notice and this permission notice shall be included
 * in all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
 * EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
 * MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
 * IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
 * CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
 * TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
 * SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
 ******************************************************************************/

package server

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

type broadcastChannel[T any] struct {
	listeners []chan T
	lock      sync.RWMutex
}

func (bc *broadcastChannel[T]) GetListener() chan T {
	bc.lock.Lock()
	defer bc.lock.Unlock()
	newChan := make(chan T)
	bc.listeners = append(bc.listeners, newChan)
	return newChan
}

func (bc *broadcastChannel[T]) RemoveListener(removeChan chan T) {
	bc.lock.Lock()
	defer bc.lock.Unlock()
	for i, listener := range bc.listeners {
		if listener == removeChan {
			bc.listeners[i] = bc.listeners[len(bc.listeners)-1]
			bc.listeners = bc.listeners[:len(bc.listeners)-1]
			close(listener)
			return
		}
	}
}

func (bc *broadcastChannel[T]) SendToAll(input T) {
	bc.lock.RLock()
	defer bc.lock.RUnlock()
	for _, listener := range bc.listeners {
		listener <- input
	}
}

func (bc *broadcastChannel[T]) runInputChannel(inputChan chan T) {
	for input := range inputChan {
		bc.SendToAll(input)
	}
}

// GetInputChannel returns a channel acting as an input to the broadcast
// channel, closing the channel will stop the worker goroutine
func (bc *broadcastChannel[T]) GetInputChannel() chan T {
	newInputChan := make(chan T)
	go bc.runInputChannel(newInputChan)
	return newInputChan
}

// ServerSentEvent is a simple struct that represents an event used in HTTP
// event stream
type ServerSentEvent struct {
	Event string
	Data  string
}

// Write will write the ServerSentEvent to the HTTP response stream and flush.
// It removes all newlines in the event data
func (sse *ServerSentEvent) Write(w http.ResponseWriter) {
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", sse.Event, strings.ReplaceAll(sse.Data, "\n", ""))
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// AddServerSentEventHandler is a shortcut for HandleServerSentEvents that
// automatically creates and returns the events channel and adds a custom
// handler for GET requests matching the provided pattern
func (s *Server) AddServerSentEventHandler(pattern string) chan *ServerSentEvent {
	eventsBroadcastChannel := broadcastChannel[*ServerSentEvent]{}

	s.router.GET(pattern, s.HandleServerSentEvents(&eventsBroadcastChannel))

	return eventsBroadcastChannel.GetInputChannel()
}

// HandleServerSentEvents is a handler function that will listen on the provided
// channel and write events to the HTTP response
func (s *Server) HandleServerSentEvents(EventsBroadcastChannel *broadcastChannel[*ServerSentEvent]) gin.HandlerFunc {
	return func(c *gin.Context) {
		w := c.Writer
		events := EventsBroadcastChannel.GetListener()
		defer EventsBroadcastChannel.RemoveListener(events)
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Content-Type", "text/event-stream")

		for {
			select {
			case e := <-events:
				e.Write(w)
			case <-c.Done():
				return
			}
		}
	}
}
