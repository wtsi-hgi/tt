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
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/wtsi-hgi/tt/database"
)

// sseThing is used to give the html string of a Thing the ability to Write()
// itself to a http.ResponseWriter as a server sent event.
type sseThing string

func (sse sseThing) Write(w http.ResponseWriter) {
	fmt.Fprintf(w, "event: newThing\ndata: %s\n\n", strings.ReplaceAll(string(sse), "\n", ""))
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

type broadcaster struct {
	listeners []chan sseThing
	sync.RWMutex
}

// Start begins ranging over the returned channel, calling Broadcast() with the
// the sseThings you send on the channel, to replicate the sseThing to all
// listeners.
func (bc *broadcaster) Start() chan sseThing {
	sseThingChan := make(chan sseThing)

	go func() {
		for input := range sseThingChan {
			bc.Broadcast(input)
		}
	}()

	return sseThingChan
}

// Broadcast replicates the given input to all listeners registerd with
// NewListener().
func (bc *broadcaster) Broadcast(input sseThing) {
	bc.RLock()
	defer bc.RUnlock()

	for _, listener := range bc.listeners {
		listener <- input
	}
}

// NewListener adds a new listener client that will receive future broadcasts
// of new sseThings.
func (bc *broadcaster) NewListener() chan sseThing {
	bc.Lock()
	defer bc.Unlock()

	newChan := make(chan sseThing)
	bc.listeners = append(bc.listeners, newChan)

	return newChan
}

// RemoveListener removes a listener channel previously supplied to
// NewListern().
func (bc *broadcaster) RemoveListener(removeChan chan sseThing) {
	bc.Lock()
	defer bc.Unlock()

	for i, listener := range bc.listeners {
		if listener == removeChan {
			bc.listeners[i] = bc.listeners[len(bc.listeners)-1]
			bc.listeners = bc.listeners[:len(bc.listeners)-1]
			close(listener)

			return
		}
	}
}

// sendNewThings creates a channel that broadcastNewThing() will send new Things
// on, which will then be replicated to every listener of this route, the
// clients getting the rendered html of the Thing.
func (s *Server) sendNewThings() func(c *gin.Context) {
	b := broadcaster{}
	s.newThingChan = b.Start()

	return func(c *gin.Context) {
		w := c.Writer
		events := b.NewListener()
		defer b.RemoveListener(events)
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

// broadcastNewThing returns an error if there's an issue rendering the given
// thing via the thing.html template. Otherwise, in a goroutine, sends the Thing
// html on the channel that replicates it to all sendNewThings() listeners.
func (s *Server) broadcastNewThing(thing *database.Thing) error {
	var renderedOutput bytes.Buffer

	err := s.rootTemplate.ExecuteTemplate(&renderedOutput, "templates/thing.html", thing)
	if err != nil {
		return err
	}

	go func() {
		s.newThingChan <- sseThing(renderedOutput.String())
	}()

	return nil
}
