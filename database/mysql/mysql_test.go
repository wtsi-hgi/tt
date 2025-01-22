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

package mysql

import (
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

const (
	filePerm = 0644
)

func TestConfig(t *testing.T) {
	Convey("Given a full set of env vars, you can make a config", t, func() {
		restore := restoreOrigEnvs()
		defer restore()

		testUser := "user"
		testPass := "pass"
		testHost := "host"
		testPort := "1234"
		testDBName := "db"

		os.Setenv(envVarUser, testUser)
		os.Setenv(envVarPass, testPass)
		os.Setenv(envVarHost, testHost)
		os.Setenv(envVarPort, testPort)
		os.Setenv(envVarDBName, testDBName)

		config, err := ConfigFromEnv()
		So(err, ShouldBeNil)
		So(config, ShouldNotBeNil)
		So(config.User, ShouldEqual, testUser)
		So(config.Passwd, ShouldEqual, testPass)
		So(config.Net, ShouldEqual, sqlNetwork)
		So(config.Addr, ShouldEqual, testHost+":"+testPort)
		So(config.DBName, ShouldEqual, testDBName)
		So(config.ParseTime, ShouldBeTrue)

		Convey("Without a full set of env vars, ConfigFromEnv fails", func() {
			os.Setenv(envVarUser, "")
			config, err := ConfigFromEnv()
			So(err, ShouldEqual, ErrMissingEnvs)
			So(config, ShouldBeNil)
		})

		Convey("You can load different environments from .env* files", func() {
			origDir, err := os.Getwd()
			So(err, ShouldBeNil)

			defer func() {
				os.Chdir(origDir)
			}()

			dir := t.TempDir()
			err = os.Chdir(dir)
			So(err, ShouldBeNil)

			err = os.WriteFile(".env.development.local",
				[]byte(envVarUser+"=devuser\n"), filePerm)
			So(err, ShouldBeNil)

			err = os.WriteFile(".env.test.local",
				[]byte(envVarUser+"=testuser\n"), filePerm)
			So(err, ShouldBeNil)

			err = os.WriteFile(".env.production.local",
				[]byte(envVarUser+"=produser\n"), filePerm)
			So(err, ShouldBeNil)

			os.Unsetenv(envVarUser)
			config, err := ConfigFromEnv()
			So(err, ShouldBeNil)
			So(config.User, ShouldEqual, "devuser")

			os.Unsetenv(envVarUser)
			os.Setenv(envVarEnv, "test")
			config, err = ConfigFromEnv()
			So(err, ShouldBeNil)
			So(config.User, ShouldEqual, "testuser")

			os.Unsetenv(envVarUser)
			os.Setenv(envVarEnv, "production")
			config, err = ConfigFromEnv()
			So(err, ShouldBeNil)
			So(config.User, ShouldEqual, "produser")

			err = os.WriteFile(".env",
				[]byte(envVarUser+"=envuser\n"+envVarDBName+"=envdb"), filePerm)
			So(err, ShouldBeNil)

			os.Unsetenv(envVarUser)
			os.Unsetenv(envVarDBName)
			os.Setenv(envVarEnv, "development")
			config, err = ConfigFromEnv()
			So(err, ShouldBeNil)
			So(config.User, ShouldEqual, "devuser")
			So(config.DBName, ShouldEqual, "envdb")

			os.Chdir(origDir)
			os.Unsetenv(envVarUser)
			os.Unsetenv(envVarDBName)
			config, err = ConfigFromEnv()
			So(err, ShouldNotBeNil)

			config, err = ConfigFromEnv(dir)
			So(err, ShouldBeNil)
			So(config.User, ShouldEqual, "devuser")
			So(config.DBName, ShouldEqual, "envdb")
		})
	})
}

// restoreOrigEnvs returns a function you should defer to restore the original
// env vars.
func restoreOrigEnvs() func() {
	origEnv := os.Getenv(envVarEnv)
	origUser := os.Getenv(envVarUser)
	origPass := os.Getenv(envVarPass)
	origHost := os.Getenv(envVarHost)
	origPort := os.Getenv(envVarPort)
	origDBName := os.Getenv(envVarDBName)

	return func() {
		os.Setenv(envVarEnv, origEnv)
		os.Setenv(envVarUser, origUser)
		os.Setenv(envVarPass, origPass)
		os.Setenv(envVarHost, origHost)
		os.Setenv(envVarPort, origPort)
		os.Setenv(envVarDBName, origDBName)
	}
}

func TestMySQL(t *testing.T) {
	restore := restoreOrigEnvs()
	defer restore()

	os.Setenv(envVarEnv, "development")
	config, err := ConfigFromEnv("../..")
	if err != nil {
		SkipConvey("Skipping MySQL tests due to missing test env vars", t, func() {})

		return
	}

	Convey("Given a config, you can connect to MySQL", t, func() {
		db, err := New(config)
		So(err, ShouldBeNil)
		So(db, ShouldNotBeNil)
	})

	Convey("Given a bad config, MySQL connections fail", t, func() {
		config.Addr = "badhost:1234"
		_, err := New(config)
		So(err, ShouldNotBeNil)
	})
}
