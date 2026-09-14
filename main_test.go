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

package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
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
)

var (
	testRootDir    string //nolint:gochecknoglobals
	testBinaryPath string //nolint:gochecknoglobals
	app            = "tt" //nolint:gochecknoglobals
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

		fmt.Println(err.Error())

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
		"-ldflags=-X github.com/wtsi-hgi/tt/cmd.Version=TEST",
		"-o", testBinaryPath,
	)

	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Printf("build failed: %s\n%s", err, strings.TrimSpace(string(out)))

		return nil
	}

	return func() { os.Remove(testBinaryPath) }
}

type testServer struct {
	key     string
	cert    string
	url     string
	stopped bool

	cmd    *exec.Cmd
	stdout *bytes.Buffer
	stderr *bytes.Buffer
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

	return net.JoinHostPort("localhost", strconv.Itoa(l.Addr().(*net.TCPAddr).Port)), nil //nolint:forcetypeassert
}

func (s *testServer) startServer(additionalServerArgs []string) {
	args := []string{"server", "--url", s.url, "--cert", s.cert, "--key", s.key}
	args = append(args, additionalServerArgs...)

	s.stopped = false
	s.cmd = exec.Command(testBinaryPath, args...) //nolint:gosec,noctx
	s.stdout = new(bytes.Buffer)
	s.stderr = new(bytes.Buffer)
	s.cmd.Stdout = s.stdout
	s.cmd.Stderr = s.stderr

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
	cmd := exec.Command(testBinaryPath, args...) //nolint:gosec,noctx

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
			return
		}

		So(err, ShouldBeNil)
		So(s.stdout.String(), ShouldBeBlank)

		waitForSomething(func() bool {
			_, err := os.Stat(logFilePath)

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

		match, err := regexp.MatchString(`^[0-9A-Za-z-]+\n$`, std)
		So(err, ShouldBeNil)
		So(match, ShouldBeTrue)
	})
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
	client := http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
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
