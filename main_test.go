/*******************************************************************************
 * Copyright (c) 2025 Genome Research Ltd.
 *
 * Author: Sendu Bala <sb10@sanger.ac.uk>
 *         Amber Faruque <af35@sanger.ac.uk>
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

package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/joho/godotenv"
	. "github.com/smartystreets/goconvey/convey"
	gas "github.com/wtsi-hgi/go-authserver"
	"github.com/wtsi-hgi/tt/database/mysql"
	"github.com/wtsi-hgi/tt/server"
)

var (
	testRootDir    string //nolint:gochecknoglobals
	testBinaryPath string //nolint:gochecknoglobals
	app            = "tt" //nolint:gochecknoglobals
)

const (
	LDAPFQDN   = "ldap-ro.internal.sanger.ac.uk"
	LDAPBindDN = "uid=%s,ou=people,dc=sanger,dc=ac,dc=uk"
)

func TestMain(m *testing.M) {
	var (
		exitCode     int
		cleanupOnce  sync.Once
		tmpRoot      string
		removeBinary func()
	)

	cleanup := func() {
		if removeBinary != nil {
			removeBinary()
		}

		if tmpRoot != "" {
			_ = os.RemoveAll(tmpRoot)
		}
	}

	sigCh := make(chan os.Signal, 2)

	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	go func() {
		<-sigCh
		cleanupOnce.Do(cleanup)
		os.Exit(130)
	}()

	defer func() {
		if rec := recover(); rec != nil {
			panic(rec)
		}

		cleanupOnce.Do(cleanup)
		os.Exit(exitCode)
	}()

	createdRoot, err := os.MkdirTemp("", "tt-tests-")
	if err != nil {
		exitCode = 1

		fmt.Println(err.Error()) //nolint:forbidigo

		return
	}

	tmpRoot = createdRoot

	testRootDir = tmpRoot

	removeBinary = buildSelf()
	if removeBinary == nil {
		return
	}

	exitCode = m.Run()
}

func buildSelf() func() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if testRootDir == "" {
		return nil
	}

	testBinaryPath = filepath.Join(testRootDir, app)

	cmd := exec.CommandContext(ctx, "go", "build", "-tags", "netgo",
		"-ldflags=-X github.com/wtsi-hgi/tt/cmd.Version=v1.0.0-gasdf",
		"-o", testBinaryPath,
	)

	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Printf("build failed: %s\n%s", err, strings.TrimSpace(string(out))) //nolint: forbidigo

		return nil
	}

	return func() { os.Remove(testBinaryPath) }
}

type lockedBuffer struct {
	sync.RWMutex
	buf bytes.Buffer
}

func (l *lockedBuffer) Write(p []byte) (int, error) {
	l.Lock()
	defer l.Unlock()

	return l.buf.Write(p)
}

func (l *lockedBuffer) String() string {
	l.RLock()
	defer l.RUnlock()

	return l.buf.String()
}

type testServer struct {
	key     string
	cert    string
	url     string
	stopped bool

	cmd    *exec.Cmd
	stdout lockedBuffer
	stderr lockedBuffer
}

func NewTestServer(t *testing.T, args []string) (*testServer, error) {
	t.Helper()

	s := new(testServer)

	envFile := ".env.development.local"

	_, err := os.Stat(envFile)
	if err != nil {
		return nil, err
	}

	err = godotenv.Load(envFile)
	if err != nil {
		return nil, err
	}

	conf, err := mysql.ConfigFromEnv(envFile)
	So(err, ShouldBeNil)

	db, err := mysql.New(conf)
	So(err, ShouldBeNil)

	So(db.Reset(), ShouldBeNil)

	s.cert, s.key, err = gas.CreateTestCert(t)
	if err != nil {
		return nil, err
	}

	s.url, err = getTestServerAddress()
	if err != nil {
		return nil, err
	}

	s.startServer(args)

	Reset(func() {
		if err := s.Shutdown(); err != nil {
			t.Errorf("server shutdown failed: %s", err)
		}
	})

	return s, nil
}

func getTestServerAddress() (string, error) {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return "", err
	}

	defer l.Close()

	return net.JoinHostPort("localhost", strconv.Itoa(l.Addr().(*net.TCPAddr).Port)), nil //nolint:forcetypeassert,errcheck
}

func (s *testServer) startServer(additionalServerArgs []string) {
	args := []string{"server", "--url", s.url, "--cert", s.cert, "--key", s.key, //nolint:goconst, prealloc
		"--domain", "@default.domain"}
	args = append(args, additionalServerArgs...)

	s.stopped = false
	s.cmd = exec.Command(testBinaryPath, args...) //nolint:noctx
	s.cmd.Stdout = &s.stdout
	s.cmd.Stderr = &s.stderr

	err := s.cmd.Start()
	So(err, ShouldBeNil)

	s.waitForServer()
}

func (s *testServer) waitForServer() {
	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		dialer := &net.Dialer{Timeout: 50 * time.Millisecond}
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		tlsDialer := tls.Dialer{NetDialer: dialer, Config: &tls.Config{InsecureSkipVerify: true}} //nolint:gosec
		conn, err := tlsDialer.DialContext(ctx, "tcp", s.url)

		cancel()

		if err == nil {
			_ = conn.Close()

			return
		}

		time.Sleep(10 * time.Millisecond)
	}
}

func (s *testServer) Shutdown() error {
	if s.stopped {
		return nil
	}

	s.stopped = true

	err := s.cmd.Process.Signal(os.Interrupt)

	errCh := make(chan error, 1)
	go func() { errCh <- s.cmd.Wait() }()

	select {
	case errb := <-errCh:
		return errors.Join(err, errb)
	case <-time.After(5 * time.Second):
		return errors.Join(err, s.cmd.Process.Kill())
	}
}

func runBinary(t *testing.T, args ...string) (int, string) {
	t.Helper()

	cmd := exec.Command(testBinaryPath, args...) //nolint:noctx

	std, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(err)
	}

	return cmd.ProcessState.ExitCode(), string(std)
}

func TestServer(t *testing.T) {
	Convey("You can start a real server from the built tt binary that logs to stderr", t, func() {
		s, err := NewTestServer(t, []string{"--logstderr"})
		if err != nil {
			SkipConvey("Skipping real server tests without .env.development.local", func() {})

			return
		}

		So(err, ShouldBeNil)
		So(s.stdout.String(), ShouldBeBlank)

		waitForSomething(func() bool {
			return s.stderr.String() != ""
		})

		So(s.stderr.String(), ShouldContainSubstring, "server started")

		Convey("You can access the webpage", func() {
			html, err := getHTML("https://" + s.url)

			So(err, ShouldBeNil)
			So(html, ShouldContainSubstring, "title>Temporary Things</title>")

			Convey("You can kill the server and it dies gracefully", func() {
				err := s.Shutdown()
				So(err, ShouldBeNil)
				So(s.stderr.String(), ShouldContainSubstring, "gracefully shut down")
			})
		})
	})

	Convey("You can start a real server that logs to a file", t, func() {
		dir := t.TempDir()
		logFilePath := filepath.Join(dir, "log")

		s, err := NewTestServer(t, []string{"--logfile", logFilePath})
		if err != nil {
			SkipConvey("Skipping real server tests without .env.development.local", func() {})

			return
		}

		So(err, ShouldBeNil)
		So(s.stdout.String(), ShouldBeBlank)

		waitForSomething(func() bool {
			_, err := os.Stat(logFilePath) //nolint:govet

			return err != nil
		})

		contents, err := os.ReadFile(logFilePath)
		So(err, ShouldBeNil)
		So(string(contents), ShouldContainSubstring, "server started")
	})

	// NB: no test for logging to syslog, because we don't have root to check...
}

func TestServerHelp(t *testing.T) {
	Convey("The server has defaults from environment variables", t, func() {
		exit, std := runBinary(t, "server", "-h")
		So(exit, ShouldBeZeroValue)
		So(std, ShouldContainSubstring, "--url")
		So(std, ShouldNotContainSubstring, "(default")

		os.Setenv("TT_SERVER_URL", "testURL")
		os.Setenv("TT_SERVER_CERT", "testCert")
		os.Setenv("TT_SERVER_KEY", "testKey")
		os.Setenv("TT_SERVER_COOKIE", "testKey")

		exit, std = runBinary(t, "server", "-h")
		So(exit, ShouldBeZeroValue)
		So(std, ShouldContainSubstring, "(default \"testURL\")")
		So(std, ShouldContainSubstring, "(default \"testCert\")")
		So(std, ShouldContainSubstring, "(default \"testKey\")")
	})
}
func TestVersion(t *testing.T) {
	Convey("The version sub command tells you the version", t, func() {
		exit, std := runBinary(t, "version")
		So(exit, ShouldBeZeroValue)
		So(std, ShouldNotBeBlank)
		match, err := regexp.MatchString(`^[0-9A-Za-z.-]+\n$`, std)
		So(err, ShouldBeNil)
		So(match, ShouldBeTrue)
	})
}

func TestCreateGet(t *testing.T) {
	Convey("You can Create a user and Create a Thing", t, func() {
		stateDir := t.TempDir()
		t.Setenv("XDG_STATE_HOME", stateDir)

		s, err := NewTestServer(t, []string{})
		if err != nil {
			SkipConvey("Skipping real server tests without .env.development.local", func() {})

			return
		}
		// user, err := user.Current()
		// So(err, ShouldBeNil)
		ec, out := runBinary(t, "user", "--url", s.url, "--cert", s.cert, "--user", "name", "--email", "name@something")
		So(out, ShouldBeBlank)
		So(ec, ShouldBeZeroValue)

		args := []string{"create", "--url", s.url, "--cert", s.cert} //nolint:prealloc,goconst

		args = append(args, createFlags(map[string]string{
			"name":          "something",        //nolint:goconst
			"version":       "1.2",              //nolint:goconst
			"description":   "Some thing",       //nolint:goconst
			"type":          "s3",               //nolint:goconst
			"requestSource": "dfdf",             //nolint:goconst
			"address":       "s3://bucket/path", //nolint:goconst
			"reason":        "just because",     //nolint:goconst
			"creator":       "name",             //nolint:goconst
		})...)

		ec, out = runBinary(t, args...)
		So(out, ShouldBeBlank)
		So(ec, ShouldBeZeroValue)

		args = []string{"create", "--url", s.url, "--cert", s.cert}
		extraCreateFlags := map[string]string{
			"name":          "something_else",
			"version":       "2.3",
			"description":   "thing of some",
			"type":          "irods",
			"requestSource": "fdfd",
			"address":       "something/path",
			"reason":        "just because",
			"creator":       "name",
		}
		argsWithExtra := append(args, createFlags(extraCreateFlags)...) //nolint: gocritic
		ec, out = runBinary(t, argsWithExtra...)
		So(out, ShouldBeBlank)
		So(ec, ShouldBeZeroValue)

		Convey("You can Create a Thing without creating the User first", func() {
			newUser := "newUser"
			extraCreateFlags["address"] = "some/other/thing/path"
			extraCreateFlags["creator"] = newUser
			newUserArgs := append(args, createFlags(extraCreateFlags)...) //nolint: gocritic
			ec, out = runBinary(t, newUserArgs...)
			So(out, ShouldBeBlank)
			So(ec, ShouldBeZeroValue)

			delete(extraCreateFlags, "creator")
			extraCreateFlags["type"] = "openstack"
			defaultUserArgs := append(args, createFlags(extraCreateFlags)...) //nolint: gocritic
			ec, out = runBinary(t, defaultUserArgs...)
			So(out, ShouldBeBlank)
			So(ec, ShouldBeZeroValue)
			Convey("You can Get the things that implemented auto creating a user", func() {
				ec, out = runBinary(t, "get", "--url", s.url, "--cert", s.cert)
				So(out, ShouldContainSubstring, newUser)
				So(out, ShouldContainSubstring, os.Getenv("USER"))
				So(ec, ShouldBeZeroValue)
			})
		})

		Convey("You can Get a Thing without filtering", func() {
			ec, out = runBinary(t, "get", "--url", s.url, "--cert", s.cert)
			So(out, ShouldContainSubstring, "something\t1.2\ts3")
			So(out, ShouldContainSubstring, "something_else\t2.3\tirods")
			So(ec, ShouldBeZeroValue)
		})

		Convey("You can Get a Thing with filtering", func() {
			args := []string{"get", "--url", s.url, "--cert", s.cert} //nolint:prealloc
			args = append(args, createFlags(map[string]string{
				"type":     "s3",
				"orderBy":  "address",
				"orderDir": "ASC",
				"page":     "1",
				"perPage":  "10",
			})...)

			ec, out = runBinary(t, args...)

			So(out, ShouldContainSubstring, "something\t1.2\ts3")
			So(out, ShouldNotContainSubstring, "something_else\t2.3\tirods")
			So(ec, ShouldBeZeroValue)
		})

		// !!
		Convey("You can't interact with protected endpoints if the server credentials are deleted", func() {
			tokenFile := filepath.Join(stateDir, ".tt.token")
			err := os.Remove(tokenFile)
			So(err, ShouldBeNil)
			err = os.Remove(filepath.Join(stateDir, ".tt.jwt"))
			So(err, ShouldBeNil)

			ec, out := runBinary(t, "user", "--url", s.url, "--cert", s.cert, "--user", "name", "--email", "name@something")
			So(out, ShouldContainSubstring, "authentication failed")

			So(ec, ShouldNotBeZeroValue)

			args := []string{"create", "--url", s.url, "--cert", s.cert} //nolint:prealloc

			args = append(args, createFlags(map[string]string{
				"name":          "something",
				"version":       "1.2",
				"description":   "Some thing",
				"type":          "s3",
				"requestSource": "dfdf",
				"address":       "s3://bucket/path",
				"reason":        "just because",
				"creator":       "name",
			})...)
			ec, out = runBinary(t, args...)
			So(out, ShouldContainSubstring, "authentication failed")
			So(ec, ShouldNotBeZeroValue)

			ec, out = runBinary(t, "get", "--url", s.url, "--cert", s.cert)
			So(out, ShouldContainSubstring, "authentication failed")
			So(ec, ShouldNotBeZeroValue)
		})

		SkipConvey("You can't create a User or a Thing if you didn't start the server, unless you are creating yourself ", func() { //nolint:lll
			newCLIUser := "root"
			origUser := os.Getenv("USER")

			os.Setenv("USER", newCLIUser)

			origHome := os.Getenv("HOME")

			os.Setenv("HOME", t.TempDir())

			origXDGState := os.Getenv("XDG_STATE_HOME")

			Reset(func() {
				os.Setenv("USER", origUser)
				os.Setenv("HOME", origHome)
				os.Setenv("XDG_STATE_HOME", origXDGState)
			})

			err := generateFakeJWT(t, s, newCLIUser)
			So(err, ShouldBeNil)

			ec, out := runBinary(t, "user", "--url", s.url, "--cert", s.cert, "--user", newCLIUser,
				"--email", newCLIUser+"@something")
			So(out, ShouldBeBlank)
			So(ec, ShouldBeZeroValue)

			u, _ := user.Current() //nolint:errcheck
			So(u.Username, ShouldContainSubstring, " ??")

			ec, out = runBinary(t, "user", "--url", s.url, "--cert", s.cert, "--user", "Not"+newCLIUser,
				"--email", "Not"+newCLIUser+"@something")
			So(out, ShouldContainSubstring, " ??")
			So(ec, ShouldBeZeroValue)
		})
		//TODO: test that we can't use protected client methods without being the user that started the server
		//  test where you change $USER and $HOME so that you are not the user that started the server: cli should fail
	})

	Convey("You can not (Create a user, Create a Thing, and Get a Thing) without starting the server", t, func() {
		testCert := filepath.Join(t.TempDir(), "testCert")
		errw := os.WriteFile(testCert, nil, 0600)
		So(errw, ShouldBeNil)

		tokenDir := t.TempDir()
		t.Setenv("XDG_STATE_HOME", tokenDir)
		tokenFile := filepath.Join(tokenDir, ".tt.token")

		errw = os.WriteFile(tokenFile, bytes.Repeat([]byte{'a'}, 44), 0600)
		So(errw, ShouldBeNil)

		ec, out := runBinary(t, "user", "--url", "testURL", "--cert", testCert, "--user", "name", "--email", "name@something") //nolint:lll // This can be args[] later
		expected := "Post \"https://testURL/rest/v1/jwt\": dial tcp: lookup testURL on 127.0.0.53:53: server misbehaving"
		So(out, ShouldContainSubstring, expected)
		So(ec, ShouldNotBeZeroValue)
	})
}
func TestServerWithCookie(t *testing.T) {
	Convey("You can start a server with cookie authentication", t, func() {
		s, err := NewTestServer(t, []string{"--cookie", "myCookieName"})
		So(err, ShouldBeNil)

		if err != nil {
			SkipConvey("Skipping real server tests without .env.development.local", func() {})

			return
		}

		Convey("You can access the things webpage given valid cookie config", func() {
			user := "any"
			cookie := &http.Cookie{ //nolint:gosec
				Name:  "myCookieName",
				Value: base64.StdEncoding.EncodeToString([]byte(user + ":")),
			}
			request, err := http.NewRequest(
				http.MethodGet,
				"https://"+s.url+server.EndPointAuthThings,
				nil,
			)
			So(err, ShouldBeNil)

			http.DefaultClient.Transport = &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, //nolint:gosec
				},
			}

			request.AddCookie(cookie)
			resp, err := http.DefaultClient.Do(request)
			So(err, ShouldBeNil)

			So(resp.StatusCode, ShouldEqual, http.StatusOK)
		})

		Convey("You can still use CLI mode to create and get things ", func() {
			ec, out := runBinary(t, "user", "--url", s.url, "--cert", s.cert, "--user", "name", "--email", "name@something")
			So(out, ShouldBeBlank)
			So(ec, ShouldBeZeroValue)

			args := []string{"create", "--url", s.url, "--cert", s.cert} //nolint:prealloc
			args = append(args, createFlags(map[string]string{
				"name":          "something",
				"version":       "1.2",
				"description":   "Some thing",
				"type":          "s3",
				"requestSource": "dfdf",
				"address":       "s3://bucket/path",
				"reason":        "just because",
				"creator":       "name",
			})...)

			ec, out = runBinary(t, args...)
			So(out, ShouldBeBlank)
			So(ec, ShouldBeZeroValue)
		})
		//TODO: maybe check non admin after fixing other non admin test..
	})
}

func createFlags(flags map[string]string) []string {
	var f []string //nolint:prealloc

	for flag, v := range flags {
		f = append(f, "--"+flag, v)
	}

	return f
}

func waitForSomething(something func() bool) {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if something() {
			break
		}

		time.Sleep(10 * time.Millisecond)
	}
}

func getHTML(url string) (string, error) {
	client := http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}} //nolint:gosec,lll

	res, err := client.Get(url)
	if err != nil {
		return "", err
	}

	html, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	res.Body.Close()

	return string(html), nil
}

// function sets some env changes and also returns original env so that we can reinstate
// this where we call it
// func (s *testServer) impersonateUser(t *testing.T, username string) ([]string, error) {
// 	t.Helper()

// 	originalEnv := slices.Clone(s.env)

// 	fakeDir, err := generateFakeJWT(t, s, username)
// 	if err != nil {
// 		return originalEnv, err
// 	}

// 	s.env = append(slices.DeleteFunc(s.env, func(str string) bool {
// 		return strings.HasPrefix(str, "XDG_STATE_HOME")
// 	}), "XDG_STATE_HOME="+fakeDir)

// 	return originalEnv, nil
// }

type mockDB struct {
	mysql.DB
}

func (mockDB) Close() error {
	return nil
}

func generateFakeJWT(t *testing.T, s *testServer, user string) error {
	t.Helper()

	fakeDir := t.TempDir()
	os.Setenv("XDG_STATE_HOME", fakeDir)

	defer os.WriteFile(filepath.Join(fakeDir, ".tt.token"), bytes.Repeat([]byte{'a'}, 44), 0600) //nolint:errcheck

	ts, err := server.New(server.Config{
		HTTPLogger:         io.Discard,
		Database:           &mockDB{},
		DefaultEmailDomain: "1",
		Key:                s.key,
		Cert:               s.cert,
		TokenFileName:      ".tt.token",
		AuthCallback:       func(username, password string) (bool, string) { return true, "0" },
	})
	if err != nil {
		return err
	}

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}

	l.Close()

	_, port, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		return err
	}

	go ts.Start("localhost:"+port, s.cert, s.key) //nolint: errcheck
	defer ts.Stop()

	time.Sleep(time.Second)

	c, err := gas.NewClientCLI(".tt.jwt", ".tt.token", "localhost:"+port, s.cert, false)
	if err != nil {
		return err
	}

	return c.Login(user, "password")
}
