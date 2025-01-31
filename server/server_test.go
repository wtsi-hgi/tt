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
	"bytes"
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	gas "github.com/wtsi-hgi/go-authserver"
	"github.com/wtsi-hgi/tt/database"
	"github.com/wtsi-hgi/tt/internal"
)

type mockDB struct {
	users    []database.User
	things   []database.Thing
	subs     []database.Subscriber
	lastPage int
}

func newMockDB() *mockDB {
	var users []database.User
	var things []database.Thing
	var subs []database.Subscriber

	return &mockDB{
		users:    users,
		things:   things,
		subs:     subs,
		lastPage: 1,
	}
}

func (m *mockDB) CreateUser(name, email string) (*database.User, error) {
	return nil, nil
}

func (m *mockDB) CreateThing(args database.CreateThingParams) (*database.Thing, error) {
	return nil, nil
}

func (m *mockDB) GetThings(params database.GetThingsParams) (*database.GetThingsResult, error) {
	return &database.GetThingsResult{
		Things:   sortAndFilterThings(m.things, params),
		LastPage: m.lastPage,
	}, nil
}

func sortAndFilterThings(origThings []database.Thing, params database.GetThingsParams) []database.Thing {
	var things []database.Thing

	order := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}

	if params.OrderBy == database.OrderByAddress {
		order = []int{4, 6, 1, 8, 2, 6, 7, 9, 3, 0}
	}

	if params.FilterOnType == database.ThingsTypeS3 {
		order = []int{4, 5}
	}

	if params.OrderDirection == database.OrderDesc {
		slices.Reverse(order)
	}

	if params.ThingsPerPage > 0 {
		low := (params.Page - 1) * params.ThingsPerPage
		if low >= len(order) {
			order = []int{}
		}
		high := low + params.ThingsPerPage
		if high > len(order) {
			high = len(order)
		}
		order = order[low:high]
	}

	for i := range origThings {
		if i >= len(order) {
			break
		}

		things = append(things, origThings[order[i]])
	}

	return things
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
		mdb := newMockDB()

		conf := Config{
			HTTPLogger: logWriter,
			Database:   mdb,
		}
		So(conf.CheckValid(), ShouldBeNil)

		s, err := New(conf)
		So(err, ShouldBeNil)

		SkipConvey("You can start and stop the server", func() {
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
			actual := testEndpoint(s, "GET", "/", nil)

			expected, err := templatesFS.ReadFile("templates/root.html")
			So(err, ShouldBeNil)

			So(actual, ShouldEqual, string(expected))
		})

		Convey("You can use the things endpoint", func() {
			actual := testEndpoint(s, "GET", "/things", nil)
			So(actual, ShouldEqual, "")

			mdb.users, mdb.things, mdb.subs = internal.GetExampleData()
			expected := executeThingsTemplate(mdb.things)

			actual = testEndpoint(s, "GET", "/things", nil)
			So(actual, ShouldEqual, expected)
			So(strings.Count(actual, "</tr"), ShouldEqual, 10)
			So(strings.Count(actual, "<td>"), ShouldEqual, 60)

			things := sortAndFilterThings(mdb.things, database.GetThingsParams{
				OrderDirection: database.OrderDesc,
			})
			expected = executeThingsTemplate(things)
			actual = testEndpoint(s, "GET", "/things?dir=DESC", nil)
			So(actual, ShouldEqual, expected)

			code := testEndpointCode(s, "GET", "/things?dir=BAD", nil)
			So(code, ShouldEqual, http.StatusBadRequest)

			things = sortAndFilterThings(mdb.things, database.GetThingsParams{
				OrderBy: database.OrderByAddress,
			})
			expected = executeThingsTemplate(things)
			actual = testEndpoint(s, "GET", "/things?sort=address", nil)
			So(actual, ShouldEqual, expected)

			code = testEndpointCode(s, "GET", "/things?sort=bad", nil)
			So(code, ShouldEqual, http.StatusBadRequest)

			things = sortAndFilterThings(mdb.things, database.GetThingsParams{
				OrderBy:        database.OrderByAddress,
				OrderDirection: database.OrderDesc,
			})
			expected = executeThingsTemplate(things)
			actual = testEndpoint(s, "GET", "/things?sort=address&dir=DESC", nil)
			So(actual, ShouldEqual, expected)

			things = sortAndFilterThings(mdb.things, database.GetThingsParams{
				FilterOnType: database.ThingsTypeS3,
			})
			expected = executeThingsTemplate(things)
			actual = testEndpoint(s, "GET", "/things?type=s3", nil)
			So(actual, ShouldEqual, expected)
			So(strings.Count(actual, "</tr>"), ShouldEqual, 2)

			code = testEndpointCode(s, "GET", "/things?type=bad", nil)
			So(code, ShouldEqual, http.StatusBadRequest)

			perPage := 3
			things = sortAndFilterThings(mdb.things, database.GetThingsParams{
				Page:          1,
				ThingsPerPage: perPage,
			})
			expected = executeThingsTemplate(things)
			actual = testEndpoint(s, "GET", "/things?page=1&per_page=3", nil)
			So(actual, ShouldEqual, expected)
			So(strings.Count(actual, "</tr>"), ShouldEqual, perPage)
			So(actual, ShouldContainSubstring, "<td>j</td>")
			So(actual, ShouldContainSubstring, "<td>c</td>")
			So(actual, ShouldContainSubstring, "<td>e</td>")

			things = sortAndFilterThings(mdb.things, database.GetThingsParams{
				Page:          2,
				ThingsPerPage: perPage,
			})
			expected = executeThingsTemplate(things)
			actual = testEndpoint(s, "GET", "/things?page=2&per_page=3", nil)
			So(actual, ShouldEqual, expected)
			So(strings.Count(actual, "</tr>"), ShouldEqual, perPage)
			So(actual, ShouldContainSubstring, "<td>i</td>")
			So(actual, ShouldContainSubstring, "<td>a</td>")
			So(actual, ShouldContainSubstring, "<td>f</td>")
		})
	})
}

func testEndpoint(s *Server, method, target string, inputBody io.Reader) string {
	recorder := recordRequest(s, method, target, inputBody)
	So(recorder.Code, ShouldEqual, http.StatusOK)
	So(recorder.Header().Get("Content-Type"), ShouldEqual, "text/html; charset=utf-8")

	return recorder.Body.String()
}

func recordRequest(s *Server, method, target string, inputBody io.Reader) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(method, target, inputBody)
	s.router.ServeHTTP(recorder, req)

	return recorder
}

func testEndpointCode(s *Server, method, target string, inputBody io.Reader) int {
	recorder := recordRequest(s, method, target, inputBody)
	return recorder.Code
}

func executeThingsTemplate(things []database.Thing) string {
	data, err := templatesFS.ReadFile("templates/things.html")
	So(err, ShouldBeNil)
	templ := template.New("")
	templChild := templ.New("templates/things.html")
	templChild, err = templChild.Parse(string(data))
	So(err, ShouldBeNil)

	data, err = templatesFS.ReadFile("templates/thing.html")
	So(err, ShouldBeNil)
	templChild = templChild.New("templates/thing.html")
	_, err = templChild.Parse(string(data))
	So(err, ShouldBeNil)

	var expectedB bytes.Buffer
	err = templ.ExecuteTemplate(&expectedB, "templates/things.html", things)
	So(err, ShouldBeNil)

	expected := expectedB.String()
	So(expected, ShouldStartWith, "\n<tr")
	So(expected, ShouldEndWith, "</tr>\n")
	So(expected, ShouldContainSubstring, "<td>")

	return expected
}
