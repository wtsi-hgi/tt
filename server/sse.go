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

	"github.com/wtsi-hgi/tt/database"
)

const sseThingsEventName = "thingsSSE"

// broadcastNewThing returns an error if there's an issue rendering the given
// thing via the thing.html template. Otherwise, in a goroutine, sends the Thing
// html on the channel that replicates it to all SSE listeners.
func (s *Server) broadcastNewThing(thing *database.Thing) error {
	var renderedOutput bytes.Buffer

	err := s.rootTemplate.ExecuteTemplate(&renderedOutput, "templates/thing.html", thing)
	if err != nil {
		return err
	}

	err = s.SSEBroadcast(sseThingsEventName, renderedOutput.String())
	s.Logger.Printf("broadcastNewThing called, got err %s sending %s", err, renderedOutput.String())
	return err
}
