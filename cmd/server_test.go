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

package cmd

import (
	"bytes"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestServer(t *testing.T) {
	Convey("Server-specific flags have sensible defaults", t, func() {
		flag := serverCmd.Flags().Lookup("logfile")
		So(flag, ShouldNotBeNil)
		So(flag.Value.String(), ShouldEqual, "")

		flag = serverCmd.Flags().Lookup("logstderr")
		So(flag, ShouldNotBeNil)
		So(flag.Value.String(), ShouldEqual, "false")
	})

	Convey("You can't start a server without all needed env vars", t, func() {
		output, err := executeRootCommandForTest(t, []string{"server"})
		So(err, ShouldNotBeNil)
		So(output, ShouldContainSubstring, "failed to get database config")
		So(output, ShouldContainSubstring, "missing required environment variables")
	})
}

func executeRootCommandForTest(t *testing.T, args []string) (string, error) {
	t.Helper()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	command := RootCmd
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs(args)

	err := command.Execute()

	return strings.TrimSpace(stdout.String() + stderr.String()), err
}
