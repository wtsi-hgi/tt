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
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	gas "github.com/wtsi-hgi/go-authserver"
	"github.com/wtsi-hgi/tt/database"
)

type mockDB struct {
}

func newMockDB() *mockDB {
	return &mockDB{}
}

func (m *mockDB) CreateUser(name, email string) (*database.User, error) {
	return nil, nil
}

func (m *mockDB) CreateThing(args database.CreateThingParams) (*database.Thing, error) {
	return nil, nil
}

func (m *mockDB) GetThings(params database.GetThingsParams) (*database.GetThingsResult, error) {
	return nil, nil
}

func (m *mockDB) DeleteUser(id uint32) error {
	return nil
}

func (m *mockDB) DeleteThing(id uint32) error {
	return nil
}

func (m *mockDB) Close() error {
	return nil
}

func TestServer(t *testing.T) {
	Convey("Given a valid Config", t, func() {
		logWriter := gas.NewStringLogger()
		database := newMockDB()

		conf := Config{
			HTTPLogger: logWriter,
			Database:   database,
		}
		So(conf.CheckValid(), ShouldBeNil)

		s, err := New(conf)
		So(err, ShouldBeNil)

		Convey("You can start and stop the server", func() {
			errCh := make(chan error, 1)

			go func() {
				errCh <- s.Start(":0")
			}()

			<-time.After(1 * time.Second)

			s.Stop()
			err = <-errCh
			So(err, ShouldBeNil)
		})

		Convey("You can use the root endpoint", func() {
			body := testEndpoint(s, "GET", "/", nil)

			data, err := templatesFS.ReadFile("templates/root.html")
			So(err, ShouldBeNil)

			So(body, ShouldEqual, string(data))
		})
	})
}

func testEndpoint(s *Server, method, target string, inputBody io.Reader) string {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(method, target, inputBody)
	s.router.ServeHTTP(recorder, req)

	So(recorder.Code, ShouldEqual, http.StatusOK)

	return recorder.Body.String()
}
