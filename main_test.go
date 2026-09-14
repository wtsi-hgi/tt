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
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
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
	testRootDir      string //nolint:gochecknoglobals
	testBinaryPath   string //nolint:gochecknoglobals
	testPSBinaryPath string //nolint:gochecknoglobals
	app              = "tt" //nolint:gochecknoglobals
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

func NewTestServer(t *testing.T) (*testServer, error) {
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

	s.startServer()

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

func (s *testServer) startServer() {
	args := []string{"server", "--url", s.url, "--cert", s.cert, "--key", s.key, "--logstderr"}

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

// func (s *testServer) runBinary(t *testing.T, args ...string) (int, string) {
// 	t.Helper()

// 	fullArgs := append([]string{"--url", s.url, "--cert", s.cert}, args...)
// 	exitCode, out := runCLI(t, s.env, "", fullArgs...)

// 	return exitCode, out
// }

// func (s *testServer) runBinaryWithNoLogging(t *testing.T, args ...string) (int, string) {
// 	t.Helper()

// 	fullArgs := append([]string{"--url", s.url, "--cert", s.cert}, args...)

// 	return runCLI(t, s.env, "", fullArgs...)
// }

// func (s *testServer) confirmOutput(t *testing.T, args []string, expectedCode int, expected string) {
// 	t.Helper()

// 	exitCode, actual := s.runBinary(t, args...)

// 	So(exitCode, ShouldEqual, expectedCode)
// 	So(actual, ShouldEqual, expected)
// }

// func (s *testServer) confirmOutputContains(t *testing.T, args []string, expectedCode int, expected string) {
// 	t.Helper()

// 	exitCode, actual := s.runBinaryWithNoLogging(t, args...)

// 	So(exitCode, ShouldEqual, expectedCode)
// 	So(actual, ShouldContainSubstring, expected)
// }

func TestServer(t *testing.T) {
	Convey("You can start a real server from the built tt binary", t, func() {
		fmt.Println(testBinaryPath)
		s, err := NewTestServer(t)
		So(err, ShouldBeNil)
		// <-time.After(5 * time.Second)
		So(s.stdout.String(), ShouldBeBlank)

		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			if s.stderr.String() != "" {
				return
			}

			time.Sleep(10 * time.Millisecond)
		}

		So(s.stderr.String(), ShouldContainSubstring, "server started")
	})
}

// TODO: test env var defaults for persistent server flags
// TODO: real server test, including kill behaviour
