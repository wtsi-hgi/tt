/*******************************************************************************
 * Copyright (c) 2025 Genome Research Ltd.
 *
 * Author: Sendu Bala <sb10@sanger.ac.uk>
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
	"embed"
	"html/template"
	"io"
	"io/fs"
	"regexp"

	gas "github.com/wtsi-hgi/go-authserver"
	"github.com/wtsi-hgi/tt/database"
)

//go:embed templates
var templatesFS embed.FS

const (
	ErrNoLogger   = gas.Error("a http logger must be configured")
	ErrNoDatabase = gas.Error("a database must be supplied")
)

// Config configures the server.
type Config struct {
	// HTTPLogger is used to log all HTTP requests. This is required.
	HTTPLogger io.Writer

	// Database is used to query MySQL for users, things and subscribers. This
	// is required.
	Database database.Queries
}

// CheckValid returns nil if all required options have been supplied, or an
// error if not.
func (c Config) CheckValid() error {
	if c.HTTPLogger == nil {
		return ErrNoLogger
	}

	if c.Database == nil {
		return ErrNoDatabase
	}

	return nil
}

// Server is used to start a web server that provides a REST API to the setdb
// package's database, and a website that displays the information nicely.
type Server struct {
	gas.Server
	db           database.Queries
	rootTemplate *template.Template
}

// New creates a Server which serves the tt website.
//
// It logs to the required configured io.Writer, which could for example be
// syslog using the log/syslog pkg with syslog.new(syslog.LOG_INFO, "tag").
func New(conf Config) (*Server, error) {
	if err := conf.CheckValid(); err != nil {
		return nil, err
	}

	s := &Server{
		Server: *gas.New(conf.HTTPLogger),
		db:     conf.Database,
	}

	s.Router().Use(gas.IncludeAbortErrorsInBody)

	err := s.addEndPoints()
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Server) addEndPoints() error {
	s.rootTemplate = template.New("")

	s.rootTemplate.Funcs(template.FuncMap{"args": func(args ...any) []any { return args }, "add": func(a, b int) int { return a + b }, "sub": func(a, b int) int { return a - b }, "rangenum": func(n int) []struct{} { return make([]struct{}, n) }})

	err := s.loadAllTemplates("templates/.*")
	if err != nil {
		return err
	}

	s.Router().SetHTMLTemplate(s.rootTemplate)

	s.Router().GET("/", s.pageRoot)
	s.Router().GET("/things", s.getThings)
	s.Router().GET("/things/listen", s.SSESender(sseThingsEventName))
	s.Router().POST("/things", s.postThing)
	s.Router().DELETE("/things/:id", s.deleteThing)

	return nil
}

func (s *Server) loadAllTemplates(pattern string) error {
	return fs.WalkDir(templatesFS, ".", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if matched, _ := regexp.MatchString(pattern, path); !d.IsDir() && matched {
			data, err := templatesFS.ReadFile(path)
			if err != nil {
				return err
			}

			t := s.rootTemplate.New(path).Funcs(s.Router().FuncMap)
			if _, err = t.Parse(string(data)); err != nil {
				return err
			}
		}

		return nil
	})
}

// Stop gracefully stops the server after Start().
func (s *Server) Stop() {
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			s.Logger.Printf("database close failed: %s", err)
		}
	}

	s.Server.Stop()

	s.Logger.Printf("gracefully shut down")
}
