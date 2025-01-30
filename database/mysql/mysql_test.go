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
	"os"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/wtsi-hgi/tt/database"
	"github.com/wtsi-hgi/tt/internal"
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
	origEnv, origEnvSet := os.LookupEnv(envVarEnv)
	origUser, origUserSet := os.LookupEnv(envVarUser)
	origPass, origPassSet := os.LookupEnv(envVarPass)
	origHost, origHostSet := os.LookupEnv(envVarHost)
	origPort, origPortSet := os.LookupEnv(envVarPort)
	origDBName, origDBNameSet := os.LookupEnv(envVarDBName)

	return func() {
		if origEnvSet {
			os.Setenv(envVarEnv, origEnv)
		} else {
			os.Unsetenv(envVarEnv)
		}

		if origUserSet {
			os.Setenv(envVarUser, origUser)
		} else {
			os.Unsetenv(envVarUser)
		}

		if origPassSet {
			os.Setenv(envVarPass, origPass)
		} else {
			os.Unsetenv(envVarPass)
		}

		if origHostSet {
			os.Setenv(envVarHost, origHost)
		} else {
			os.Unsetenv(envVarHost)
		}

		if origPortSet {
			os.Setenv(envVarPort, origPort)
		} else {
			os.Unsetenv(envVarPort)
		}

		if origDBNameSet {
			os.Setenv(envVarDBName, origDBName)
		} else {
			os.Unsetenv(envVarDBName)
		}
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
				expectedUsers, expectedThings, expectedSubs := internal.GetExampleData()

				user1, err := db.CreateUser(expectedUsers[0].Name, expectedUsers[0].Email)
				So(err, ShouldBeNil)
				So(user1, ShouldResemble, &expectedUsers[0])

				user2, err := db.CreateUser(expectedUsers[1].Name, expectedUsers[1].Email)
				So(err, ShouldBeNil)
				So(user2, ShouldResemble, &expectedUsers[1])

				_, err = db.CreateUser(expectedUsers[0].Name, "foo@bar.com")
				So(err, ShouldNotBeNil)

				_, err = db.CreateUser("foo", expectedUsers[1].Email)
				So(err, ShouldNotBeNil)

				count, err = countTableRows(db.pool, "users")
				So(err, ShouldBeNil)
				So(count, ShouldEqual, 2)

				for i, et := range expectedThings {
					before := time.Now()

					var creator database.User

					if expectedSubs[i].UserID == expectedUsers[0].ID {
						creator = expectedUsers[0]
					} else {
						creator = expectedUsers[1]
					}

					thing, err := db.CreateThing(database.CreateThingParams{
						Address:     et.Address,
						Type:        et.Type,
						Description: et.Description,
						Reason:      et.Reason,
						Remove:      et.Remove,
						Creator:     creator.Name,
					})
					So(err, ShouldBeNil)

					after := time.Now()
					created := thing.Created
					So(created, ShouldHappenOnOrBetween, before, after)

					thing.Created = time.Time{}
					So(thing, ShouldResemble, &et)
				}

				_, err = db.CreateThing(database.CreateThingParams{
					Address:     "addr",
					Type:        database.ThingsTypeIrods,
					Description: "desc",
					Reason:      "reason",
					Remove:      expectedThings[0].Remove,
					Creator:     "invalid",
				})
				So(err, ShouldNotBeNil)
				So(err, ShouldEqual, ErrNoUser)

				numThings := len(expectedThings)
				count, err = countTableRows(db.pool, "things")
				So(err, ShouldBeNil)
				So(count, ShouldEqual, numThings)

				count, err = countTableRows(db.pool, "subscribers")
				So(err, ShouldBeNil)
				So(count, ShouldEqual, numThings)

				count, err = countTableRows(db.pool, "subscribers", "creator = 1")
				So(err, ShouldBeNil)
				So(count, ShouldEqual, numThings)

				count, err = countTableRows(db.pool, "subscribers", "user_id = 1")
				So(err, ShouldBeNil)
				So(count, ShouldEqual, numThings/2)

				count, err = countTableRows(db.pool, "subscribers", "user_id = 2")
				So(err, ShouldBeNil)
				So(count, ShouldEqual, numThings/2)

				Convey("Then you can get things with desired sorting, pagination and filtering", func() {
					result, err := db.GetThings(database.GetThingsParams{})
					So(err, ShouldBeNil)
					So(len(result.Things), ShouldEqual, numThings)
					So(result.LastPage, ShouldEqual, 0)
					result.Things[0].Created = time.Time{}
					So(result.Things[0], ShouldResemble, expectedThings[0])
					result.Things[numThings-1].Created = time.Time{}
					So(result.Things[numThings-1], ShouldResemble, expectedThings[numThings-1])

					result, err = db.GetThings(database.GetThingsParams{
						OrderDirection: database.OrderDesc,
					})
					So(err, ShouldBeNil)
					So(len(result.Things), ShouldEqual, numThings)
					So(result.Things[0].Address, ShouldEqual, expectedThings[numThings-1].Address)
					So(result.Things[0].Type, ShouldEqual, expectedThings[numThings-1].Type)
					So(result.Things[0].Reason, ShouldEqual, expectedThings[numThings-1].Reason)
					So(result.Things[0].Remove.Format(time.DateOnly), ShouldEqual, "1979-01-02")
					So(result.Things[numThings-1].Remove.Format(time.DateOnly), ShouldEqual, "1970-01-02")

					result, err = db.GetThings(database.GetThingsParams{
						OrderBy: database.OrderByAddres,
					})
					So(err, ShouldBeNil)
					So(len(result.Things), ShouldEqual, numThings)
					So(result.Things[0].Address, ShouldEqual, "a")
					So(result.Things[numThings-1].Address, ShouldEqual, "j")

					result, err = db.GetThings(database.GetThingsParams{
						OrderBy:        database.OrderByAddres,
						OrderDirection: database.OrderDesc,
					})
					So(err, ShouldBeNil)
					So(len(result.Things), ShouldEqual, numThings)
					So(result.Things[0].Address, ShouldEqual, "j")
					So(result.Things[numThings-1].Address, ShouldEqual, "a")

					result, err = db.GetThings(database.GetThingsParams{
						OrderBy: database.OrderByType,
					})
					So(err, ShouldBeNil)
					So(len(result.Things), ShouldEqual, numThings)
					So(result.Things[0].Type, ShouldEqual, database.ThingsTypeDir)
					So(result.Things[1].Type, ShouldEqual, database.ThingsTypeDir)
					So(result.Things[2].Type, ShouldEqual, database.ThingsTypeFile)
					So(result.Things[3].Type, ShouldEqual, database.ThingsTypeFile)
					So(result.Things[4].Type, ShouldEqual, database.ThingsTypeIrods)
					So(result.Things[5].Type, ShouldEqual, database.ThingsTypeIrods)
					So(result.Things[6].Type, ShouldEqual, database.ThingsTypeOpenstack)
					So(result.Things[7].Type, ShouldEqual, database.ThingsTypeOpenstack)
					So(result.Things[8].Type, ShouldEqual, database.ThingsTypeS3)
					So(result.Things[9].Type, ShouldEqual, database.ThingsTypeS3)

					result, err = db.GetThings(database.GetThingsParams{
						OrderBy: database.OrderByReason,
					})
					So(err, ShouldBeNil)
					So(len(result.Things), ShouldEqual, numThings)
					So(result.Things[0].Reason, ShouldEqual, "a")
					So(result.Things[numThings-1].Reason, ShouldEqual, "j")

					result, err = db.GetThings(database.GetThingsParams{
						OrderBy: database.OrderByRemove,
					})
					So(err, ShouldBeNil)
					So(len(result.Things), ShouldEqual, numThings)
					So(result.Things[0].Remove.Format(time.DateOnly), ShouldEqual, "1970-01-02")
					So(result.Things[numThings-1].Remove.Format(time.DateOnly), ShouldEqual, "1979-01-02")

					result, err = db.GetThings(database.GetThingsParams{
						FilterOnType: database.ThingsTypeNil,
					})
					So(err, ShouldBeNil)
					So(len(result.Things), ShouldEqual, numThings)

					result, err = db.GetThings(database.GetThingsParams{
						FilterOnType: database.ThingsTypeIrods,
					})
					So(err, ShouldBeNil)
					So(len(result.Things), ShouldEqual, 2)
					So(result.Things[0].Type, ShouldEqual, database.ThingsTypeIrods)
					So(result.Things[1].Type, ShouldEqual, database.ThingsTypeIrods)

					page := 1
					perPage := 3
					result, err = db.GetThings(database.GetThingsParams{
						Page:          page,
						ThingsPerPage: perPage,
					})
					So(err, ShouldBeNil)
					So(result.LastPage, ShouldEqual, 4)
					So(len(result.Things), ShouldEqual, perPage)
					So(result.Things[0].ID, ShouldEqual, 1)
					So(result.Things[perPage-1].ID, ShouldEqual, 3)

					page++
					result, err = db.GetThings(database.GetThingsParams{
						Page:          page,
						ThingsPerPage: perPage,
					})
					So(err, ShouldBeNil)
					So(result.LastPage, ShouldEqual, 4)
					So(len(result.Things), ShouldEqual, perPage)
					So(result.Things[0].ID, ShouldEqual, 4)
					So(result.Things[perPage-1].ID, ShouldEqual, 6)

					page++
					result, err = db.GetThings(database.GetThingsParams{
						Page:          page,
						ThingsPerPage: perPage,
					})
					So(err, ShouldBeNil)
					So(result.LastPage, ShouldEqual, 4)
					So(len(result.Things), ShouldEqual, perPage)
					So(result.Things[0].ID, ShouldEqual, 7)
					So(result.Things[perPage-1].ID, ShouldEqual, 9)

					page++
					result, err = db.GetThings(database.GetThingsParams{
						Page:          page,
						ThingsPerPage: perPage,
					})
					So(err, ShouldBeNil)
					So(result.LastPage, ShouldEqual, 4)
					So(len(result.Things), ShouldEqual, 1)
					So(result.Things[0].ID, ShouldEqual, 10)

					page++
					result, err = db.GetThings(database.GetThingsParams{
						Page:          page,
						ThingsPerPage: perPage,
					})
					So(err, ShouldBeNil)
					So(result.LastPage, ShouldEqual, 4)
					So(len(result.Things), ShouldEqual, 0)

					result, err = db.GetThings(database.GetThingsParams{
						OrderBy:        database.OrderByReason,
						OrderDirection: database.OrderDesc,
						FilterOnType:   database.ThingsTypeS3,
						Page:           2,
						ThingsPerPage:  1,
					})
					So(err, ShouldBeNil)
					So(result.LastPage, ShouldEqual, 2)
					So(len(result.Things), ShouldEqual, 1)
					So(result.Things[0].ID, ShouldEqual, 5)
				})

				Convey("Then you can delete users and things", func() {
					err = db.DeleteUser(2)
					So(err, ShouldBeNil)

					count, err = countTableRows(db.pool, "users")
					So(err, ShouldBeNil)
					So(count, ShouldEqual, 1)

					count, err = countTableRows(db.pool, "things")
					So(err, ShouldBeNil)
					So(count, ShouldEqual, numThings)

					count, err = countTableRows(db.pool, "subscribers")
					So(err, ShouldBeNil)
					So(count, ShouldEqual, numThings/2)

					err = db.DeleteThing(3)
					So(err, ShouldBeNil)

					count, err = countTableRows(db.pool, "users")
					So(err, ShouldBeNil)
					So(count, ShouldEqual, 1)

					count, err = countTableRows(db.pool, "things")
					So(err, ShouldBeNil)
					So(count, ShouldEqual, numThings-1)

					count, err = countTableRows(db.pool, "subscribers")
					So(err, ShouldBeNil)
					So(count, ShouldEqual, (numThings/2)-1)
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

func countTableRows(pool *sql.DB, table string, where ...string) (int64, error) {
	var count int64

	sql := "SELECT COUNT(*) FROM " + table

	if len(where) == 1 {
		sql += " WHERE " + where[0]
	}

	row := pool.QueryRow(sql)
	err := row.Scan(&count)

	return count, err
}
