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
)

const (
	serverTokenBasename = ".tt.token"
)

func TestClient(t *testing.T) {
	Convey("Given a started Server and connected Client", t, func() {
		serverURL, err := getTestServerAddress()
		So(err, ShouldBeNil)

		serverCert, serverKey, err := gas.CreateTestCert(t)
		mdb := newMockDB()
		logWriter := gas.NewStringLogger()

		conf := Config{
			HTTPLogger: logWriter,
			Database:   mdb,
		}

		s, err := New(conf)
		So(err, ShouldBeNil)

		err = s.EnableAuthWithServerToken(serverCert, serverKey, serverTokenBasename, func(username, password string) (bool, string) {
			return false, ""
		})
		So(err, ShouldBeNil)

		errCh := make(chan error)
		go func() {
			err := s.Start(serverURL, serverCert, serverKey)
			errCh <- err
		}()

		defer func() {
			err = <-errCh
			So(err, ShouldBeNil)
		}()

		defer s.Stop()

		defer t.Setenv("XDG_STATE_HOME", os.Getenv("XDG_STATE_HOME"))

		fakeDir := t.TempDir()

		t.Setenv("XDG_STATE_HOME", fakeDir)

		// proabably our NewClient doesn't take a jwt,						what is meant by this?
		// There are 2 possibilities. Either we have to call gas.NewClientCLI ourselves and Login(),
		// which creates the jwt, then we supply it to NewClient() as the 3rd arg,
		// or we have a more friendly system where NewClient() does all of that stuff for us and we
		// don't worry about having to do that setup. For our testsing purposes, though, I guess we
		// need to be in control of the login. But it would mean passing username and password to NewClient
		// instead of the JWT.

		// so moving the ibackup fakejwt functionality to NewClient() and pass
		// in user and password in order to complete that login Yes; the right
		// approach depends on who else will call NewClient() (other than this
		// test), but it's a good starting point. We can change it back to take
		// a JWT later if this turns out to be problematic

		time.Sleep(1 * time.Second) // need to wait for the server to start before trying to connect to it with a client

		_, err = NewClient(serverURL, serverCert)
		So(err, ShouldNotBeNil)

		// would be good to test NewClient() now

		// Convey("You can create and get a User", func() {
		// 	userPost := &database.User{} //TODO: fill in some test details

		// 	err := client.CreateUser(userPost)
		// 	So(err, ShouldBeNil)

		// 	// our implementation does something like this:
		// 	// client := gas.NewClientRequest(url, cert)
		// 	// resp, err := client.SetBody(&userPost).Post("/user")
		// 	// if err != nil {
		// 	// 	return err
		// 	// }
		// 	// if resp.StatusCode() != http.StatusNoContent {
		// 	// 	return errors.New(resp.String()) //nolint:err113
		// 	// }

		// 	u, err := client.GetUser(userPost.Name)
		// 	So(err, ShouldBeNil)
		// 	So(u, ShouldResemble, userPost)
		// })

		// Convey("You can create and get Things", func() {
		// 	// TODO
		// })
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
