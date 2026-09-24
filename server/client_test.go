/*******************************************************************************
 * Copyright (c) 2026 Genome Research Ltd.
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
	"net"
	"os"
	"strconv"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	gas "github.com/wtsi-hgi/go-authserver"
	"github.com/wtsi-hgi/tt/database"
)

const (
	serverTokenBasename = ".tt.token"
)

func TestClient(t *testing.T) {
	Convey("Given a started Server and connected Client", t, func() {
		serverURL, err := getTestServerAddress()
		So(err, ShouldBeNil)

		serverCert, serverKey, err := gas.CreateTestCert(t)
		So(err, ShouldBeNil)

		mdb := newMockDB()
		logWriter := gas.NewStringLogger()

		conf := Config{
			HTTPLogger: logWriter,
			Database:   mdb,
		}

		s, err := New(conf)
		So(err, ShouldBeNil)

		err = s.EnableAuthWithServerToken(serverCert, serverKey, serverTokenBasename,
			func(username, password string) (bool, string) {
				return true, ""
			})
		So(err, ShouldBeNil)

		errCh := make(chan error)

		go func() {
			errc := s.Start(serverURL, serverCert, serverKey)
			errCh <- errc
		}()

		defer func() {
			err = <-errCh
			So(err, ShouldBeNil)
		}()

		defer s.Stop()

		defer t.Setenv("XDG_STATE_HOME", os.Getenv("XDG_STATE_HOME"))

		fakeDir := t.TempDir()

		t.Setenv("XDG_STATE_HOME", fakeDir)

		time.Sleep(1 * time.Second)

		client, err := NewClient(serverURL, serverCert, "user", "pass")
		So(err, ShouldBeNil)

		Convey("You can create and get a User", func() {
			userPost := &database.User{
				Name:  "User1",
				Email: "User1@something",
			}

			err := client.CreateUser(userPost)
			So(err, ShouldBeNil)

			So(len(mdb.users), ShouldEqual, 1)

			u, err := client.GetUserByName(userPost.Name)
			So(err, ShouldBeNil)
			So(u, ShouldResemble, userPost)
		})

		Convey("You can create and get Things", func() {
			thingPost := &database.Thing{
				Name:          "something_else",
				Version:       "2.3",
				Description:   "thing of some",
				Type:          "irods",
				RequestSource: "fdfd",
				Address:       "something/path",
				Reason:        "just because"}
			deleteTimeBefore := time.Now().Add(fiveYear)
			err := client.PostThing(thingPost)
			deleteTimeAfter := time.Now().Add(fiveYear)

			So(err, ShouldBeNil)
			So(len(mdb.things), ShouldEqual, 1)

			thingTwoPost := &database.Thing{
				ID:            1,
				Name:          "something_else",
				Version:       "2.3",
				Description:   "thing of some",
				Type:          "s3",
				RequestSource: "fdfd",
				Address:       "something/path",
				Reason:        "just because"}

			err = client.PostThing(thingTwoPost)
			So(err, ShouldBeNil)
			So(len(mdb.things), ShouldEqual, 2)

			//TODO: should test get with and without filtering.

			filter := &database.GetThingsParams{
				FilterOnType:   database.ThingsTypeIrods,
				OrderBy:        database.OrderByType,
				OrderDirection: database.OrderAsc,
				Page:           1,
				ThingsPerPage:  1,
			}

			things, err := client.GetThings(filter)
			So(err, ShouldBeNil)
			So(len(things), ShouldEqual, 1)
			So(things[0].Remove, ShouldHappenOnOrBetween, deleteTimeBefore, deleteTimeAfter)

			things[0].Remove = time.Time{}
			So(things[0], ShouldResemble, *thingPost)

			// Getting both with no Type filter.
			filter = &database.GetThingsParams{
				FilterOnType:   database.ThingsTypeNil,
				OrderBy:        database.OrderByType,
				OrderDirection: database.OrderAsc,
				Page:           1,
				ThingsPerPage:  2,
			}

			things, err = client.GetThings(filter)
			So(err, ShouldBeNil)
			So(len(things), ShouldEqual, 2)
			things[0].Remove = time.Time{}
			things[1].Remove = time.Time{}
			So(things[0], ShouldResemble, *thingPost)
			So(things[1], ShouldResemble, *thingTwoPost)
		})
	})
}

func getTestServerAddress() (string, error) {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return "", err
	}

	defer l.Close()

	return net.JoinHostPort("localhost", strconv.Itoa(l.Addr().(*net.TCPAddr).Port)), nil //nolint:forcetypeassert,errcheck
}
