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
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/wtsi-hgi/tt/database/types"
)

const (
	filePerm      = 0644
	envVarDoTests = "TT_SQL_DO_TESTS"
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

			os.Unsetenv(envVarEnv)
			os.Unsetenv(envVarUser)
			_, err = ConfigFromEnv()
			So(err, ShouldNotBeNil)

			os.Unsetenv(envVarEnv)
			os.Unsetenv(envVarUser)
			os.Setenv(envVarEnv, "development")
			config, err = ConfigFromEnv()
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

			os.Unsetenv(envVarUser)
			os.Unsetenv(envVarDBName)
			os.Unsetenv(envVarEnv)
			config, err = ConfigFromEnv()
			So(err, ShouldBeNil)
			So(config.User, ShouldEqual, "envuser")
			So(config.DBName, ShouldEqual, "envdb")

			os.Chdir(origDir)
			os.Unsetenv(envVarUser)
			os.Unsetenv(envVarDBName)
			_, err = ConfigFromEnv()
			So(err, ShouldNotBeNil)

			config, err = ConfigFromEnv(dir)
			So(err, ShouldBeNil)
			So(config.User, ShouldEqual, "envuser")
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
	if os.Getenv(envVarDoTests) != "TABLES_WILL_BE_DROPPED" || err != nil {
		SkipConvey("Skipping MySQL tests due to missing test env vars", t, func() {})

		return
	}

	Convey("Given a config, you can connect to MySQL", t, func() {
		db, err := New(config)
		So(err, ShouldBeNil)
		So(db, ShouldNotBeNil)

		Convey("You can reset the database", func() {
			err = db.Reset()
			So(err, ShouldBeNil)

			count, err := countTableRows(db.pool, "things")
			So(err, ShouldBeNil)
			So(count, ShouldEqual, 0)

			count, err = countTableRows(db.pool, "users")
			So(err, ShouldBeNil)
			So(count, ShouldEqual, 0)

			count, err = countTableRows(db.pool, "subscribers")
			So(err, ShouldBeNil)
			So(count, ShouldEqual, 0)

			Convey("You can then add users and things", func() {
				emailSuffix := "@example.com"
				u1 := "user1"
				u2 := "user2"

				user1, err := db.CreateUser(u1, u1+emailSuffix)
				So(err, ShouldBeNil)
				So(user1, ShouldResemble, &types.User{
					ID:    1,
					Name:  u1,
					Email: u1 + emailSuffix,
				})

				user2, err := db.CreateUser(u2, u2+emailSuffix)
				So(err, ShouldBeNil)
				So(user2, ShouldResemble, &types.User{
					ID:    2,
					Name:  u2,
					Email: u2 + emailSuffix,
				})

				_, err = db.CreateUser(u1, u1+"@foo.com")
				So(err, ShouldNotBeNil)

				_, err = db.CreateUser("foo", u1+emailSuffix)
				So(err, ShouldNotBeNil)

				count, err = countTableRows(db.pool, "users")
				So(err, ShouldBeNil)
				So(count, ShouldEqual, 2)

				var numThings uint32

				year := uint32(1970)

				var expectedThings []types.Thing

				for _, thingType := range types.ThingsTypes() {
					for j := range 2 {
						numThings++

						creator := user1
						if j%2 != 0 {
							creator = user2
						}

						address := fmt.Sprintf("%s://%d", thingType, numThings)
						description := fmt.Sprintf("desc %d", numThings)
						reason := fmt.Sprintf("reason %d", numThings)

						remove, err := time.Parse(time.DateOnly, fmt.Sprintf("%d-01-02", year+numThings))
						So(err, ShouldBeNil)

						before := time.Now()

						thing, err := db.CreateThing(types.CreateThingParams{
							Address:     address,
							Type:        thingType,
							Creator:     creator,
							Description: description,
							Reason:      reason,
							Remove:      remove,
						})
						So(err, ShouldBeNil)

						after := time.Now()
						created := thing.Created
						So(created, ShouldHappenOnOrBetween, before, after)

						thing.Created = time.Time{}

						expectedThing := types.Thing{
							ID:          numThings,
							Address:     address,
							Type:        thingType,
							Creator:     creator,
							Description: description,
							Reason:      reason,
							Remove:      remove,
						}

						expectedThings = append(expectedThings, expectedThing)

						So(thing, ShouldResemble, &expectedThing)
					}
				}

				count, err = countTableRows(db.pool, "things")
				So(err, ShouldBeNil)
				So(count, ShouldEqual, numThings)

				Convey("Then you can get things with desired sorting, pagination and filtering", func() {
					things, err := db.GetThings(types.GetThingsParams{})
					So(err, ShouldBeNil)
					So(len(things), ShouldEqual, numThings)
					things[0].Created = time.Time{}
					So(things[0], ShouldResemble, expectedThings[0])
					things[numThings-1].Created = time.Time{}
					So(things[numThings-1], ShouldResemble, expectedThings[numThings-1])

					things, err = db.GetThings(types.GetThingsParams{
						OrderDirection: types.OrderDesc,
					})
					So(err, ShouldBeNil)
					So(len(things), ShouldEqual, numThings)
					So(things[0].Address, ShouldEqual, "s3://10")
					So(things[numThings-1].Address, ShouldEqual, "dir://1")
				})
			})
		})
	})

	Convey("Given a bad config, MySQL connections fail", t, func() {
		config.Addr = "badhost:1234"
		_, err := New(config)
		So(err, ShouldNotBeNil)
	})
}

func countTableRows(pool *sql.DB, table string) (int64, error) {
	var count int64

	row := pool.QueryRow("SELECT COUNT(*) FROM " + table)
	err := row.Scan(&count)

	return count, err
}
