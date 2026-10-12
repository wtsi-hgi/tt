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
	"encoding/base64"
	"encoding/json"
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"os/user"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/gin-gonic/gin"
	. "github.com/smartystreets/goconvey/convey"
	gas "github.com/wtsi-hgi/go-authserver"
	"github.com/wtsi-hgi/tt/database"
	"github.com/wtsi-hgi/tt/internal"
	"gopkg.in/tylerb/graceful.v1"
)

const (
	ErrInvalidUserID       = database.Error("No User found with that ID")
	defaultTestEmailDomain = "@default.domain"
)

type mockDB struct {
	users    []*database.User
	things   []database.Thing
	subs     []database.Subscriber
	thingID  uint32
	lastPage int
}

func newMockDB() *mockDB {
	var (
		users  []*database.User
		things []database.Thing
		subs   []database.Subscriber
	)

	return &mockDB{
		users:    users,
		things:   things,
		subs:     subs,
		lastPage: 1,
	}
}

func (m *mockDB) CreateUser(name, email string) (*database.User, error) {
	user := &database.User{
		ID:    uint32(len(m.users) + 1), //nolint:gosec
		Name:  name,
		Email: email,
	}
	m.users = append(m.users, user)

	return user, nil
}

func (m *mockDB) GetUserByName(name string) (*database.User, error) {
	for _, user := range m.users {
		if name == user.Name {
			return user, nil
		}
	}

	return nil, ErrNoUser
}

func (m *mockDB) GetUserByID(id uint32) (*database.User, error) {
	for _, user := range m.users {
		if id == user.ID {
			return user, nil
		}
	}

	return nil, ErrInvalidUserID
}

func (m *mockDB) GetThingByID(id uint32) (*database.Thing, error) {
	for _, thing := range m.things {
		if id == thing.ID {
			return &thing, nil
		}
	}

	return nil, ErrInvalidThingID
}

func (m *mockDB) CreateThing(args database.CreateThingParams) (*database.Thing, error) {
	id := m.thingID
	m.thingID++

	thing := database.Thing{
		ID:             id,
		Address:        args.Address,
		Type:           args.Type,
		Description:    args.Description,
		Reason:         args.Reason,
		Remove:         args.Remove,
		License:        args.License,
		Version:        args.Version,
		Name:           args.Name,
		URL:            args.URL,
		DownloadMethod: args.DownloadMethod,
		RequestSource:  args.RequestSource,
		CreationDate:   args.CreationDate,
	}

	var userID uint32

	found := false

	for _, user := range m.users {
		if user.Name == args.Creator {
			userID = user.ID
			found = true
		}
	}

	if !found {
		user, err := m.CreateUser(args.Creator, args.Creator+defaultTestEmailDomain)
		if err != nil {
			return nil, err
		}

		userID = user.ID
	}

	subscriber := database.Subscriber{
		UserID:  userID,
		ThingID: id,
		Creator: true,
	}

	m.subs = append(m.subs, subscriber)
	m.things = append(m.things, thing)

	return &thing, nil
}

func (m *mockDB) GetThings(params database.GetThingsParams) (*database.GetThingsResult, error) {
	return &database.GetThingsResult{
		Things:   sortAndFilterThings(m.things, params),
		LastPage: m.lastPage,
	}, nil
}

func sortAndFilterThings(origThings []database.Thing, params database.GetThingsParams) []database.Thing {
	var things []database.Thing

	for _, thing := range origThings {
		if params.FilterOnType == database.ThingsTypeNil || thing.Type == params.FilterOnType {
			things = append(things, thing)
		}
	}

	switch params.OrderBy {
	case database.OrderByType:
		slices.SortFunc(things, func(a, b database.Thing) int {
			return strings.Compare(string(a.Type), string(b.Type))
		})
	case database.OrderByAddress:
		slices.SortFunc(things, func(a, b database.Thing) int {
			return strings.Compare(a.Address, b.Address)
		})
	case database.OrderByReason:
		slices.SortFunc(things, func(a, b database.Thing) int {
			return strings.Compare(a.Reason, b.Reason)
		})
	case database.OrderByRemove:
		slices.SortFunc(things, func(a, b database.Thing) int {
			return a.Remove.Compare(b.Remove)
		})
	}

	if params.OrderDirection == database.OrderDesc {
		slices.Reverse(things)
	}

	if params.ThingsPerPage == 0 {
		return things
	}

	low := (params.Page - 1) * params.ThingsPerPage
	high := params.Page * params.ThingsPerPage

	if low > len(things) {
		return nil
	}

	return things[low:min(high, len(things))]
}

func (m *mockDB) DeleteUser(id uint32) error {
	return nil
}

func (m *mockDB) GetSubscribers(id uint32) ([]database.Subscriber, error) {
	subscribers := []database.Subscriber{}

	for _, sub := range m.subs {
		if id == sub.ThingID {
			subscribers = append(subscribers, sub)
		}
	}

	return subscribers, nil
}

func (m *mockDB) DeleteThing(id uint32) error {
	for i, thing := range m.things {
		if thing.ID == id {
			m.things = append(m.things[:i], m.things[i+1:]...)
		}
	}

	return nil
}

func (m *mockDB) Close() error {
	return nil
}

func TestServer(t *testing.T) {
	Convey("Given a valid Config", t, func() {
		logWriter := gas.NewStringLogger()
		mdb := newMockDB()
		serverCert, serverKey, err := gas.CreateTestCert(t)
		So(err, ShouldBeNil)

		conf := Config{
			HTTPLogger:         logWriter,
			Database:           mdb,
			DefaultEmailDomain: defaultTestEmailDomain,
			AuthCallback:       func(username, password string) (bool, string) { return true, "1" },
			Key:                serverKey,
			Cert:               serverCert,
			TokenFileName:      serverTokenBasename,
		}
		So(conf.CheckValid(), ShouldBeNil)

		s, err := New(conf)
		So(err, ShouldBeNil)

		us := (*struct {
			_          *gin.Engine
			_          *graceful.Server
			_          sync.Mutex
			_          *gin.RouterGroup
			serverUser string
		})(unsafe.Pointer(&s.Server))
		w := httptest.NewRecorder()
		user, err := user.Current()
		So(err, ShouldBeNil)

		r, err := http.NewRequest(http.MethodPost, gas.EndPointJWT,
			strings.NewReader("username="+user.Username+"&password=123"))
		So(err, ShouldBeNil)

		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		s.Router().ServeHTTP(w, r)

		var jwt string

		So(json.Unmarshal(w.Body.Bytes(), &jwt), ShouldBeNil)

		Convey("You can start and stop the server", func() {
			errCh := make(chan error, 1)

			go func() {
				errCh <- s.Start(":0", "", "")
			}()

			<-time.After(1 * time.Second)

			s.Stop()

			err = <-errCh
			So(err, ShouldBeNil)
		})

		Convey("You can use the root endpoint", func() {
			actual := testEndpoint(s, "GET", pageRootPath, jwt) //!! does it need to be EndPointAuthRoot? broke a test when was

			expected, err := templatesFS.ReadFile("templates/root.html")
			So(err, ShouldBeNil)

			So(actual, ShouldEqual, string(expected))
		})

		Convey("You can GET the things endpoint", func() {
			actual := testEndpoint(s, "GET", EndPointAuthThings, jwt)
			So(actual, ShouldEqual, "")

			mdb.users, mdb.things, mdb.subs = internal.GetExampleData()

			expected := executeThingsTemplate(mdb.things)

			actual = testEndpoint(s, "GET", EndPointAuthThings, jwt)
			So(actual, ShouldEqual, expected)
			So(strings.Count(actual, "</tr"), ShouldEqual, 10)
			So(strings.Count(actual, "<td>"), ShouldEqual, 60)

			things := sortAndFilterThings(mdb.things, database.GetThingsParams{
				OrderDirection: database.OrderDesc,
			})
			expected = executeThingsTemplate(things)
			actual = testEndpoint(s, "GET", EndPointAuthThings+"?dir=DESC", jwt)
			So(actual, ShouldEqual, expected)

			code := testEndpointCode(s, "GET", EndPointAuthThings+"?dir=BAD", "", jwt)
			So(code, ShouldEqual, http.StatusBadRequest)

			things = sortAndFilterThings(mdb.things, database.GetThingsParams{
				OrderBy: database.OrderByAddress,
			})
			expected = executeThingsTemplate(things)
			actual = testEndpoint(s, "GET", EndPointAuthThings+"?sort=address", jwt)
			So(actual, ShouldEqual, expected)

			code = testEndpointCode(s, "GET", EndPointAuthThings+"?sort=bad", "", jwt)
			So(code, ShouldEqual, http.StatusBadRequest)

			things = sortAndFilterThings(mdb.things, database.GetThingsParams{
				OrderBy:        database.OrderByAddress,
				OrderDirection: database.OrderDesc,
			})
			expected = executeThingsTemplate(things)
			actual = testEndpoint(s, "GET", EndPointAuthThings+"?sort=address&dir=DESC", jwt)
			So(actual, ShouldEqual, expected)

			things = sortAndFilterThings(mdb.things, database.GetThingsParams{
				FilterOnType: database.ThingsTypeS3,
			})
			expected = executeThingsTemplate(things)
			actual = testEndpoint(s, "GET", EndPointAuthThings+"?type=s3", jwt)
			So(actual, ShouldEqual, expected)
			So(strings.Count(actual, "</tr>"), ShouldEqual, 2)

			code = testEndpointCode(s, "GET", EndPointAuthThings+"?type=bad", "", jwt)
			So(code, ShouldEqual, http.StatusBadRequest)

			perPage := 3
			things = sortAndFilterThings(mdb.things, database.GetThingsParams{
				Page:          1,
				ThingsPerPage: perPage,
			})
			expected = executeThingsTemplate(things)
			actual = testEndpoint(s, "GET", EndPointAuthThings+"?page=1&per_page=3", jwt)
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
			actual = testEndpoint(s, "GET", EndPointAuthThings+"?page=2&per_page=3", jwt)
			So(actual, ShouldEqual, expected)
			So(strings.Count(actual, "</tr>"), ShouldEqual, perPage)
			So(actual, ShouldContainSubstring, "<td>i</td>")
			So(actual, ShouldContainSubstring, "<td>a</td>")
			So(actual, ShouldContainSubstring, "<td>f</td>")
		})

		Convey("You can POST to the user endpoint", func() {
			user := internal.GetExampleUser(2)
			thingJSON, err := json.Marshal(user)
			So(err, ShouldBeNil)

			code := testEndpointCode(s, "POST", EndPointAuthUser, string(thingJSON), jwt)
			So(code, ShouldEqual, http.StatusNoContent)

			So(len(mdb.users), ShouldEqual, 1)

			user.ID = 1
			So(mdb.users[0], ShouldResemble, user)

			Convey("You can POST thing details to the things endpoint and listen for SSE updates", func() {
				newUser, thing, exampleSubscriber := internal.GetExampleResourceData()
				thingParams := thing.ToCreateParams(newUser)

				thing.ID = 0
				thingJSON, err = json.Marshal(thingParams)
				So(err, ShouldBeNil)

				code := testEndpointCode(s, "POST", EndPointAuthThings, string(thingJSON), jwt)

				So(code, ShouldEqual, http.StatusNoContent)
				So(len(mdb.things), ShouldEqual, 1)
				So(mdb.things[0], ShouldResemble, thing)
				Convey("You are a subscriber to the thing you created", func() {
					newUser.ID = 2
					newUser.Email = "user1" + s.DefaultEmailDomain

					So(len(mdb.users), ShouldEqual, 2)
					So(mdb.users[1], ShouldResemble, newUser)

					exampleSubscriber.UserID = newUser.ID
					exampleSubscriber.ThingID = 0

					So(len(mdb.subs), ShouldEqual, 1)
					So(mdb.subs[0], ShouldResemble, exampleSubscriber)
				})

				//TODO: actual SSE test
			})
		})

		Convey("You can't POST to the things endpoint without providing complete thing details", func() {
			code := testEndpointCode(s, "POST", EndPointAuthThings, "", jwt)
			So(code, ShouldEqual, http.StatusBadRequest)
			So(len(mdb.things), ShouldEqual, 0)
		})

		Convey("You can not POST another user to the user endpoint if you are not an admin", func() {
			us.serverUser = "Admin"
			user := internal.GetExampleUser(2)
			thingJSON, err := json.Marshal(user)
			So(err, ShouldBeNil)

			code := testEndpointCode(s, "POST", EndPointAuthUser, string(thingJSON), jwt)
			So(code, ShouldEqual, http.StatusUnauthorized)
			So(len(mdb.things), ShouldEqual, 0)

			Convey("You can not POST to the things endpoint if you are not an admin", func() {
				newUser, thing, _ := internal.GetExampleResourceData()
				thingParams := thing.ToCreateParams(newUser)

				thing.ID = 0
				thingJSON, err = json.Marshal(thingParams)
				So(err, ShouldBeNil)

				code := testEndpointCode(s, "POST", EndPointAuthThings, string(thingJSON), jwt)
				So(code, ShouldEqual, http.StatusUnauthorized)
				So(len(mdb.things), ShouldEqual, 0)
			})
		})

		Convey("You can POST yourself user to the user endpoint as a non Admin", func() {
			us.serverUser = "Admin"
			currentUser := user
			user := &database.User{
				ID:    1,
				Name:  currentUser.Username,
				Email: currentUser.Username + conf.DefaultEmailDomain,
			}
			thingJSON, err := json.Marshal(user)
			So(err, ShouldBeNil)

			code := testEndpointCode(s, "POST", EndPointAuthUser, string(thingJSON), jwt)
			So(code, ShouldEqual, http.StatusNoContent)

			So(len(mdb.users), ShouldEqual, 1)

			user.ID = 1
			So(mdb.users[0], ShouldResemble, user)

			Convey("You can POST your things to the thing endpoint ", func() {
				thingPostAndSubTests(mdb, s, jwt)
			})
		})

		Convey("You can not post things without a valid authentication method", func() {
			code := testEndpointCode(s, "POST", EndPointAuthThings, "", 0)
			So(code, ShouldEqual, http.StatusUnauthorized)
			So(len(mdb.things), ShouldEqual, 0)
		})
	})
}

func TestWebServerWithCookie(t *testing.T) {
	Convey("Given a valid Config", t, func() {
		logWriter := gas.NewStringLogger()
		mdb := newMockDB()
		serverCert, serverKey, err := gas.CreateTestCert(t)
		So(err, ShouldBeNil)

		conf := Config{
			HTTPLogger:         logWriter,
			Database:           mdb,
			DefaultEmailDomain: defaultTestEmailDomain,
			Key:                serverKey,
			Cert:               serverCert,
			TokenFileName:      serverTokenBasename,
			AuthCallback:       func(username, password string) (bool, string) { return true, "1" },
			CookieName:         "nginxauth",
		}

		s, err := New(conf)
		So(err, ShouldBeNil)

		user := "any"
		cookie := &http.Cookie{ //nolint:gosec
			Name:  "nginxauth",
			Value: base64.StdEncoding.EncodeToString([]byte(user + ":")),
		}

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(
			http.MethodGet,
			EndPointAuthThings,
			nil,
		)
		request.AddCookie(cookie)
		s.Router().ServeHTTP(recorder, request)
		So(recorder.Code, ShouldEqual, http.StatusOK)

		Convey("You can start and stop the server", func() {
			errCh := make(chan error, 1)

			go func() {
				errCh <- s.Start(":0", "", "")
			}()

			<-time.After(1 * time.Second)

			s.Stop()

			err = <-errCh
			So(err, ShouldBeNil)
		})

		Convey("You can use the root endpoint", func() {
			actual := testEndpoint(s, "GET", pageRootPath, cookie)

			expected, err := templatesFS.ReadFile("templates/root.html")
			So(err, ShouldBeNil)

			So(actual, ShouldEqual, string(expected))
		})

		Convey("You can GET the things endpoint", func() {
			actual := testEndpoint(s, "GET", EndPointAuthThings, cookie)
			So(actual, ShouldEqual, "")

			mdb.users, mdb.things, mdb.subs = internal.GetExampleData()

			expected := executeThingsTemplate(mdb.things)

			actual = testEndpoint(s, "GET", EndPointAuthThings, cookie)
			So(actual, ShouldEqual, expected)
			So(strings.Count(actual, "</tr"), ShouldEqual, 10)
			So(strings.Count(actual, "<td>"), ShouldEqual, 60)

			things := sortAndFilterThings(mdb.things, database.GetThingsParams{
				OrderDirection: database.OrderDesc,
			})
			expected = executeThingsTemplate(things)
			actual = testEndpoint(s, "GET", EndPointAuthThings+"?dir=DESC", cookie)
			So(actual, ShouldEqual, expected)

			code := testEndpointCode(s, "GET", EndPointAuthThings+"?dir=BAD", "", cookie)
			So(code, ShouldEqual, http.StatusBadRequest)

			things = sortAndFilterThings(mdb.things, database.GetThingsParams{
				OrderBy: database.OrderByAddress,
			})
			expected = executeThingsTemplate(things)
			actual = testEndpoint(s, "GET", EndPointAuthThings+"?sort=address", cookie)
			So(actual, ShouldEqual, expected)

			code = testEndpointCode(s, "GET", EndPointAuthThings+"?sort=bad", "", cookie)
			So(code, ShouldEqual, http.StatusBadRequest)
		})

		Convey("You can not POST to the user endpoint, unless you are posting yourself", func() {
			userPost := internal.GetExampleUser(2)
			thingJSON, err := json.Marshal(userPost)
			So(err, ShouldBeNil)

			code := testEndpointCode(s, "POST", EndPointAuthUser, string(thingJSON), cookie)
			So(code, ShouldEqual, http.StatusUnauthorized)

			So(len(mdb.users), ShouldEqual, 0)

			userPost = &database.User{
				ID:    1,
				Name:  user,
				Email: user + conf.DefaultEmailDomain,
			}
			thingJSON, err = json.Marshal(userPost)
			So(err, ShouldBeNil)

			code = testEndpointCode(s, "POST", EndPointAuthUser, string(thingJSON), cookie)
			So(code, ShouldEqual, http.StatusNoContent)

			So(len(mdb.users), ShouldEqual, 1)

			userPost.ID = 1
			So(mdb.users[0], ShouldResemble, userPost)
			Convey("You can POST your things to the thing endpoint ", func() {
				thingPostAndSubTests(mdb, s, cookie)
			})
			//TODO: actual SSE test
		})

		Convey("You can POST a thing to the thing endpoint without being a user yet", func() {
			So(len(mdb.users), ShouldEqual, 0)

			_, thing, _ := internal.GetExampleResourceData()
			thingParams := thing.ToCreateParams(&database.User{Name: user})

			thing.ID = 0
			thingJSON, err := json.Marshal(thingParams)
			So(err, ShouldBeNil)

			code := testEndpointCode(s, "POST", EndPointAuthThings, string(thingJSON), cookie)

			So(code, ShouldEqual, http.StatusNoContent)
			So(len(mdb.things), ShouldEqual, 1)
			So(mdb.things[0], ShouldResemble, thing)
			So(len(mdb.users), ShouldEqual, 1)

			Convey("You are a subscriber to the thing you created", func() {
				exampleSubscriber := database.Subscriber{
					UserID:  1,
					ThingID: thing.ID,
					Creator: true,
				}

				So(len(mdb.subs), ShouldEqual, 1)
				So(mdb.subs[0], ShouldResemble, exampleSubscriber)
			})
		})
	})
}

func thingPostAndSubTests(mdb *mockDB, s *Server, auth any) {
	_, thing, _ := internal.GetExampleResourceData()
	thingParams := thing.ToCreateParams(mdb.users[0])

	thing.ID = 0
	thingJSON, err := json.Marshal(thingParams)
	So(err, ShouldBeNil)

	code := testEndpointCode(s, "POST", EndPointAuthThings, string(thingJSON), auth)

	So(code, ShouldEqual, http.StatusNoContent)
	So(len(mdb.things), ShouldEqual, 1)
	So(mdb.things[0], ShouldResemble, thing)

	Convey("You are a subscriber to the thing you created", func() {
		exampleSubscriber := database.Subscriber{
			UserID:  1,
			ThingID: thing.ID,
			Creator: true,
		}

		So(len(mdb.subs), ShouldEqual, 1)
		So(mdb.subs[0], ShouldResemble, exampleSubscriber)
	})
}

func testEndpoint(s *Server, method, target string, auth any) string { //nolint:unparam
	recorder := recordRequest(s, method, target, "", auth)
	So(recorder.Code, ShouldEqual, http.StatusOK)
	So(recorder.Header().Get("Content-Type"), ShouldEqual, "text/html; charset=utf-8")

	return recorder.Body.String()
}

func recordRequest(s *Server, method, target, body string, auth any) *httptest.ResponseRecorder {
	var bodyReader io.Reader

	if body != "" {
		bodyReader = strings.NewReader(body)
	}

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(method, target, bodyReader)

	switch auth := auth.(type) {
	case *http.Cookie:
		req.AddCookie(auth)
	case string:
		req.Header.Set("Authorization", "Bearer "+auth)
	}

	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	s.Router().ServeHTTP(recorder, req)

	return recorder
}

func testEndpointCode(s *Server, method, target, body string, auth any) int {
	recorder := recordRequest(s, method, target, body, auth)

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
