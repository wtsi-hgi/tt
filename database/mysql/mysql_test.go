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

				i := uint32(0)
				year := uint32(1970)
				thingsTypes := []types.ThingsType{
					types.ThingsTypeIrods,
					types.ThingsTypeDir,
					types.ThingsTypeS3,
					types.ThingsTypeFile,
					types.ThingsTypeOpenstack,
				}
				thingsPerType := 2
				numThings := len(thingsTypes) * thingsPerType
				expectedThings := make([]types.Thing, numThings)
				addresses := []string{
					"j", "c", "e", "i", "a", "f", "b", "g", "d", "h",
				}
				reasons := []string{
					"i", "c", "g", "e", "a", "d", "f", "h", "j", "b",
				}

				for _, thingType := range thingsTypes {
					for j := range thingsPerType {
						creator := user1
						if j%2 != 0 {
							creator = user2
						}

						remove, err := time.Parse(time.DateOnly, fmt.Sprintf("%d-01-02", year+i))
						So(err, ShouldBeNil)

						before := time.Now()

						thing, err := db.CreateThing(types.CreateThingParams{
							Address:     addresses[i],
							Type:        thingType,
							Creator:     creator,
							Description: "desc",
							Reason:      reasons[i],
							Remove:      remove,
						})
						So(err, ShouldBeNil)

						after := time.Now()
						created := thing.Created
						So(created, ShouldHappenOnOrBetween, before, after)

						expectedThing := types.Thing{
							ID:          i + 1,
							Address:     addresses[i],
							Type:        thingType,
							Creator:     creator,
							Description: "desc",
							Reason:      reasons[i],
							Remove:      remove,
						}
						expectedThings[i] = expectedThing

						thing.Created = time.Time{}
						So(thing, ShouldResemble, &expectedThing)

						i++
					}
				}

				count, err = countTableRows(db.pool, "things")
				So(err, ShouldBeNil)
				So(count, ShouldEqual, numThings)

				Convey("Then you can get things with desired sorting, pagination and filtering", func() {
					result, err := db.GetThings(types.GetThingsParams{})
					So(err, ShouldBeNil)
					So(len(result.Things), ShouldEqual, numThings)
					So(result.LastPage, ShouldEqual, 0)
					result.Things[0].Created = time.Time{}
					So(result.Things[0], ShouldResemble, expectedThings[0])
					result.Things[numThings-1].Created = time.Time{}
					So(result.Things[numThings-1], ShouldResemble, expectedThings[numThings-1])

					result, err = db.GetThings(types.GetThingsParams{
						OrderDirection: types.OrderDesc,
					})
					So(err, ShouldBeNil)
					So(len(result.Things), ShouldEqual, numThings)
					So(result.Things[0].Address, ShouldEqual, addresses[numThings-1])
					So(result.Things[0].Type, ShouldEqual, thingsTypes[len(thingsTypes)-1])
					So(result.Things[0].Reason, ShouldEqual, reasons[numThings-1])
					So(result.Things[0].Remove.Format(time.DateOnly), ShouldEqual, "1979-01-02")
					So(result.Things[numThings-1].Remove.Format(time.DateOnly), ShouldEqual, "1970-01-02")

					result, err = db.GetThings(types.GetThingsParams{
						OrderBy: types.OrderByAddres,
					})
					So(err, ShouldBeNil)
					So(len(result.Things), ShouldEqual, numThings)
					So(result.Things[0].Address, ShouldEqual, "a")
					So(result.Things[numThings-1].Address, ShouldEqual, "j")

					result, err = db.GetThings(types.GetThingsParams{
						OrderBy:        types.OrderByAddres,
						OrderDirection: types.OrderDesc,
					})
					So(err, ShouldBeNil)
					So(len(result.Things), ShouldEqual, numThings)
					So(result.Things[0].Address, ShouldEqual, "j")
					So(result.Things[numThings-1].Address, ShouldEqual, "a")

					result, err = db.GetThings(types.GetThingsParams{
						OrderBy: types.OrderByType,
					})
					So(err, ShouldBeNil)
					So(len(result.Things), ShouldEqual, numThings)
					So(result.Things[0].Type, ShouldEqual, types.ThingsTypeFile)
					So(result.Things[1].Type, ShouldEqual, types.ThingsTypeFile)
					So(result.Things[2].Type, ShouldEqual, types.ThingsTypeDir)
					So(result.Things[3].Type, ShouldEqual, types.ThingsTypeDir)
					So(result.Things[4].Type, ShouldEqual, types.ThingsTypeIrods)
					So(result.Things[5].Type, ShouldEqual, types.ThingsTypeIrods)
					So(result.Things[6].Type, ShouldEqual, types.ThingsTypeS3)
					So(result.Things[7].Type, ShouldEqual, types.ThingsTypeS3)
					So(result.Things[8].Type, ShouldEqual, types.ThingsTypeOpenstack)
					So(result.Things[9].Type, ShouldEqual, types.ThingsTypeOpenstack)

					result, err = db.GetThings(types.GetThingsParams{
						OrderBy: types.OrderByReason,
					})
					So(err, ShouldBeNil)
					So(len(result.Things), ShouldEqual, numThings)
					So(result.Things[0].Reason, ShouldEqual, "a")
					So(result.Things[numThings-1].Reason, ShouldEqual, "j")

					result, err = db.GetThings(types.GetThingsParams{
						OrderBy: types.OrderByRemove,
					})
					So(err, ShouldBeNil)
					So(len(result.Things), ShouldEqual, numThings)
					So(result.Things[0].Remove.Format(time.DateOnly), ShouldEqual, "1970-01-02")
					So(result.Things[numThings-1].Remove.Format(time.DateOnly), ShouldEqual, "1979-01-02")

					result, err = db.GetThings(types.GetThingsParams{
						FilterOnType: types.ThingsTypeIrods,
					})
					So(err, ShouldBeNil)
					So(len(result.Things), ShouldEqual, thingsPerType)
					So(result.Things[0].Type, ShouldEqual, types.ThingsTypeIrods)
					So(result.Things[1].Type, ShouldEqual, types.ThingsTypeIrods)

					page := 1
					perPage := 3
					result, err = db.GetThings(types.GetThingsParams{
						Page:          page,
						ThingsPerPage: perPage,
					})
					So(err, ShouldBeNil)
					So(result.LastPage, ShouldEqual, 4)
					So(len(result.Things), ShouldEqual, perPage)
					So(result.Things[0].ID, ShouldEqual, 1)
					So(result.Things[perPage-1].ID, ShouldEqual, 3)

					page++
					result, err = db.GetThings(types.GetThingsParams{
						Page:          page,
						ThingsPerPage: perPage,
					})
					So(err, ShouldBeNil)
					So(result.LastPage, ShouldEqual, 4)
					So(len(result.Things), ShouldEqual, perPage)
					So(result.Things[0].ID, ShouldEqual, 4)
					So(result.Things[perPage-1].ID, ShouldEqual, 6)

					page++
					result, err = db.GetThings(types.GetThingsParams{
						Page:          page,
						ThingsPerPage: perPage,
					})
					So(err, ShouldBeNil)
					So(result.LastPage, ShouldEqual, 4)
					So(len(result.Things), ShouldEqual, perPage)
					So(result.Things[0].ID, ShouldEqual, 7)
					So(result.Things[perPage-1].ID, ShouldEqual, 9)

					page++
					result, err = db.GetThings(types.GetThingsParams{
						Page:          page,
						ThingsPerPage: perPage,
					})
					So(err, ShouldBeNil)
					So(result.LastPage, ShouldEqual, 4)
					So(len(result.Things), ShouldEqual, 1)
					So(result.Things[0].ID, ShouldEqual, 10)

					page++
					result, err = db.GetThings(types.GetThingsParams{
						Page:          page,
						ThingsPerPage: perPage,
					})
					So(err, ShouldBeNil)
					So(result.LastPage, ShouldEqual, 4)
					So(len(result.Things), ShouldEqual, 0)

					result, err = db.GetThings(types.GetThingsParams{
						OrderBy:        types.OrderByReason,
						OrderDirection: types.OrderDesc,
						FilterOnType:   types.ThingsTypeS3,
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
					So(count, ShouldEqual, numThings/2)

					err = db.DeleteThing(3)
					So(err, ShouldBeNil)

					count, err = countTableRows(db.pool, "users")
					So(err, ShouldBeNil)
					So(count, ShouldEqual, 1)

					count, err = countTableRows(db.pool, "things")
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

func countTableRows(pool *sql.DB, table string) (int64, error) {
	var count int64

	row := pool.QueryRow("SELECT COUNT(*) FROM " + table)
	err := row.Scan(&count)

	return count, err
}
