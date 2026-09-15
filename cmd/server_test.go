/*******************************************************************************
 * Copyright (c) 2025 Genome Research Ltd.
 *
 * Author: Sendu Bala <sb10@sanger.ac.uk>
 *         Amber Faruque<af35@sanger.ac.uk>
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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joho/godotenv"
	. "github.com/smartystreets/goconvey/convey"
	gas "github.com/wtsi-hgi/go-authserver"
	ttmysql "github.com/wtsi-hgi/tt/database/mysql"
)

func TestServer(t *testing.T) {
	Convey("Given a server", t, func() {
		Convey("Server-specific flags have sensible defaults", func() {
			flag := serverCmd.Flags().Lookup("logfile")
			So(flag, ShouldNotBeNil)
			So(flag.Value.String(), ShouldEqual, "")

			flag = serverCmd.Flags().Lookup("logstderr")
			So(flag, ShouldNotBeNil)
			So(flag.Value.String(), ShouldEqual, "false")
		})

		Reset(func() {
			flags := serverCmd.Flags()

			for _, name := range []string{"url", "cert", "key"} {
				flag := flags.Lookup(name)
				if flag == nil {
					continue
				}

				flags.Set(name, flag.DefValue) //nolint:errcheck
				flag.Changed = false
			}
		})

		Convey("You can't start a server without all needed env vars", func() {
			output, err := executeRootCommandForTest(t, []string{"server", "--cert", "a", "--url", "w", "--key", "b"}) //nolint:goconst,lll
			So(err, ShouldNotBeNil)
			So(output, ShouldContainSubstring, "failed to get database config")
			So(output, ShouldContainSubstring, "missing required environment variables")
		})

		Convey("Given needed env vars", func() {
			const envVarVal = "val"
			os.Setenv(ttmysql.EnvVarHost, envVarVal)
			os.Setenv(ttmysql.EnvVarPort, envVarVal)
			os.Setenv(ttmysql.EnvVarUser, envVarVal)
			os.Setenv(ttmysql.EnvVarPass, envVarVal)
			os.Setenv(ttmysql.EnvVarDBName, envVarVal)

			cliArgs := []string{"server"}

			output, err := executeRootCommandForTest(t, cliArgs)
			So(err, ShouldNotBeNil)
			So(output, ShouldNotContainSubstring, "missing required environment variables")

			Convey("You can't start a server without --url", func() {
				So(output, ShouldContainSubstring, `"cert", "key", "url" not set`)

				Convey("Given URL", func() {
					cliArgs = append(cliArgs, "--url", envVarVal)
					output, err = executeRootCommandForTest(t, cliArgs)
					So(err, ShouldNotBeNil)
					So(output, ShouldNotContainSubstring, `"url"`)

					Convey("You can't start a server without --cert", func() {
						So(output, ShouldContainSubstring, `"cert", "key" not set`)

						Convey("Given cert", func() {
							cliArgs = append(cliArgs, "--cert", envVarVal)
							output, err = executeRootCommandForTest(t, cliArgs)
							So(err, ShouldNotBeNil)
							So(output, ShouldNotContainSubstring, `"cert"`)

							Convey("You can't start a server without --key", func() {
								So(output, ShouldContainSubstring, `"key" not set`)

								Convey("Given key", func() {
									cliArgs = append(cliArgs, "--key", envVarVal)
									output, err = executeRootCommandForTest(t, cliArgs)
									So(err, ShouldNotBeNil)
									So(output, ShouldNotContainSubstring, `"key"`)

									Convey("You can't start a server without valid db info", func() {
										So(output, ShouldContainSubstring, "error opening database")

										dir, err := os.Getwd()

										So(err, ShouldBeNil)

										parentDir := filepath.Dir(dir)
										envFile := filepath.Join(parentDir, ".env.development.local")

										_, err = os.Stat(envFile)
										if err != nil {
											SkipConvey("Skipping real server tests without "+envFile, func() {})

											return
										}

										os.Unsetenv(ttmysql.EnvVarHost)
										os.Unsetenv(ttmysql.EnvVarPort)
										os.Unsetenv(ttmysql.EnvVarUser)
										os.Unsetenv(ttmysql.EnvVarPass)
										os.Unsetenv(ttmysql.EnvVarDBName)

										err = godotenv.Load(envFile)
										if err != nil {
											SkipConvey(fmt.Sprintf("Skipping real server tests due to error reading file: %s", err), func() {})

											return
										}

										Convey("Given real database details", func() {
											Convey("You can't start a server with invalid cert files", func() {
												output, err := executeRootCommandForTest(t, cliArgs)
												So(err, ShouldNotBeNil)
												So(output, ShouldContainSubstring, "non-graceful stop: open val: no such file or directory")
											})

											Convey("Given real cert files", func() {
												cert, key, err := gas.CreateTestCert(t)
												So(err, ShouldBeNil)

												Convey("You can't start a server with an invalid url", func() {
													cliArgs = []string{"server", "--url", "invalid", "--cert", cert, "--key", key}
													output, err := executeRootCommandForTest(t, cliArgs)
													So(err, ShouldNotBeNil)
													So(output, ShouldContainSubstring, "non-graceful stop: listen tcp: address invalid: missing port in address") //nolint:lll
												})
											})
										})
									})
								})
							})
						})
					})
				})
			})
		})
	})
}

func executeRootCommandForTest(t *testing.T, args []string) (string, error) {
	t.Helper()

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	command := RootCmd
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs(args)

	err := command.Execute()

	return strings.TrimSpace(stdout.String() + stderr.String()), err
}
