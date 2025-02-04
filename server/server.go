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
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	gas "github.com/wtsi-hgi/go-authserver"
	"github.com/wtsi-hgi/tt/database"
	"gopkg.in/tylerb/graceful.v1"
)

//go:embed templates
var templatesFS embed.FS

const (
	ErrNoLogger   = gas.Error("a http logger must be configured")
	ErrNoDatabase = gas.Error("a database must be supplied")

	stopTimeout       = 10 * time.Second
	readHeaderTimeout = 20 * time.Second
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
	router       *gin.Engine
	srv          *graceful.Server
	srvMutex     sync.Mutex
	db           database.Queries
	Logger       *log.Logger
	rootTemplate *template.Template
	newThingChan chan sseThing
}

// New creates a Server which serves the tt website.
//
// It logs to the required configured io.Writer, which could for example be
// syslog using the log/syslog pkg with syslog.new(syslog.LOG_INFO, "tag").
func New(conf Config) (*Server, error) {
	if err := conf.CheckValid(); err != nil {
		return nil, err
	}

	//TODO: reimplement Server by embedding gas.Server (from which much of this
	// implementation was copy/pasted from), when we're ready to implement and
	// require authentication

	gin.SetMode(gin.ReleaseMode)

	r := gin.New()

	logger := log.New(conf.HTTPLogger, "", 0)

	gin.DisableConsoleColor()
	gin.DefaultWriter = logger.Writer()

	r.Use(ginLogger())

	r.Use(gin.RecoveryWithWriter(conf.HTTPLogger))

	s := &Server{
		router: r,
		db:     conf.Database,
		Logger: logger,
	}

	s.router.Use(gas.IncludeAbortErrorsInBody)

	err := s.addEndPoints()
	if err != nil {
		return nil, err
	}

	return s, nil
}

// ginLogger returns a handler that will format logs in a way that is searchable
// and nice in syslog output.
func ginLogger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s - [%s %s %s \"%s\"] STATUS=%d %s %s\n",
			param.ClientIP,
			param.Method,
			param.Path,
			param.Request.Proto,
			param.Request.UserAgent(),
			param.StatusCode,
			param.Latency,
			param.ErrorMessage,
		)
	})
}

func (s *Server) addEndPoints() error {
	s.rootTemplate = template.New("")

	err := s.loadAllTemplates("templates/.*")
	if err != nil {
		return err
	}

	s.router.SetHTMLTemplate(s.rootTemplate)

	s.router.GET("/", s.pageRoot)
	s.router.GET("/things", s.getThings)
	s.router.GET("/things/listen", s.sendNewThings())
	s.router.POST("/things", s.postThing)
	s.router.DELETE("/things/:id", s.deleteThing)

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

			t := s.rootTemplate.New(path).Funcs(s.router.FuncMap)
			if _, err = t.Parse(string(data)); err != nil {
				return err
			}
		}

		return nil
	})
}

// Start starts listening on the given addr, blocking until Stop() is called
// (in another goroutine), or until a SIGINT or SIGTERM is received.
func (s *Server) Start(addr string) error {
	srv := &graceful.Server{
		Timeout: stopTimeout,

		Server: &http.Server{
			Addr:              addr,
			Handler:           s.router,
			ReadHeaderTimeout: readHeaderTimeout,
		},
	}

	s.srvMutex.Lock()
	s.srv = srv
	s.srvMutex.Unlock()

	return srv.ListenAndServe()
}

// Stop gracefully stops the server after Start().
func (s *Server) Stop() {
	s.srvMutex.Lock()

	if s.srv == nil {
		s.srvMutex.Unlock()

		return
	}

	srv := s.srv
	s.srv = nil

	if s.db != nil {
		if err := s.db.Close(); err != nil {
			s.Logger.Printf("database close failed: %s", err)
		}
	}

	s.srvMutex.Unlock()

	close(s.newThingChan)

	ch := srv.StopChan()
	srv.Stop(stopTimeout)
	<-ch

	s.Logger.Printf("gracefully shut down")
}
