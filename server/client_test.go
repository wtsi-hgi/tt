/*******************************************************************************
 * Copyright (c) 2026 Genome Research Ltd.
 *
 * Authors:
*	- Sendu Bala <sb10@sanger.ac.uk>
*   - Amber Faruque <af35@sanger.ac.uk>
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
	"os"
	"testing"
	"time"

	"github.com/guregu/null/v5"
	. "github.com/smartystreets/goconvey/convey"
	gas "github.com/wtsi-hgi/go-authserver"
	"github.com/wtsi-hgi/tt/database"
)

func TestClient(t *testing.T) {
	Convey("Given a started Server and connected Client", t, func() {
		home := os.Getenv("XDG_STATE_HOME")

		Reset(func() { t.Setenv("XDG_STATE_HOME", home) })

		fakeDir := t.TempDir()

		t.Setenv("XDG_STATE_HOME", fakeDir)

		serverCert, serverKey, err := gas.CreateTestCert(t)
		So(err, ShouldBeNil)

		mdb := newMockDB()
		logWriter := gas.NewStringLogger()

		conf := Config{
			HTTPLogger:         logWriter,
			Database:           mdb,
			DefaultEmailDomain: defaultTestEmailDomain,
			AuthCallback:       func(username, password string) (bool, string) { return true, "1" },
			Key:                serverKey,
			Cert:               serverCert,
			TokenFileName:      serverTokenBasename,
		}

		s, err := New(conf)
		So(err, ShouldBeNil)

		serverURL, stopServerFunc, err := gas.StartTestServer(s, serverCert, serverKey)
		defer stopServerFunc() //nolint:errcheck

		So(err, ShouldBeNil)
		client, err := NewClient(serverURL, serverCert)
		So(err, ShouldBeNil)

		Convey("You can create and get a User", func() {
			userPost := &database.User{
				Name:  "User1",
				Email: "User1" + s.DefaultEmailDomain,
			}
			err := client.CreateUser(userPost)
			So(err, ShouldBeNil)

			So(len(mdb.users), ShouldEqual, 1)

			u, err := client.GetUserByName(userPost.Name)
			So(err, ShouldBeNil)

			userPost.ID = 1
			So(u, ShouldResemble, userPost)

			u, err = client.GetUserByID(u.ID)
			So(err, ShouldBeNil)
			So(u, ShouldResemble, userPost)
		})

		Convey("You can create and get Things", func() {
			userName := "testCreator"

			thingPost := &database.CreateThingParams{
				Name:          "something_else",
				Version:       "2.3",
				Description:   "thing of some",
				Type:          "irods",
				RequestSource: "fdfd",
				Address:       "something/path",
				Reason:        "just because",
				Creator:       userName,
			}
			deleteTimeBefore := time.Now().Add(fiveYear)
			err := client.PostThing(thingPost)
			deleteTimeAfter := time.Now().Add(fiveYear)

			So(err, ShouldBeNil)
			So(len(mdb.things), ShouldEqual, 1)

			before := time.Now()
			thingTwoPost := &database.CreateThingParams{
				Name:          "something_else",
				Version:       "2.3",
				Description:   "thing of some",
				Type:          "s3",
				RequestSource: "fdfd",
				Address:       "something/path",
				Reason:        "just because",
				Creator:       userName,
				CreationDate:  null.TimeFrom(before),
			}

			time.Sleep(time.Second)

			err = client.PostThing(thingTwoPost)

			So(err, ShouldBeNil)
			So(len(mdb.things), ShouldEqual, 2)

			// Get irods post and verify deletion time automated without creation date.
			filter := &database.GetThingsParams{
				FilterOnType:   database.ThingsTypeIrods,
				OrderBy:        database.OrderByType,
				OrderDirection: database.OrderAsc,
				Page:           1,
				ThingsPerPage:  1,
			}

			things, err := client.GetThings(filter)
			So(err, ShouldBeNil)
			So(len(things), ShouldEqual, 1)
			So(things[0].Remove, ShouldHappenOnOrBetween, deleteTimeBefore, deleteTimeAfter)

			things[0].Remove = time.Time{}
			So(things[0].ToCreateParams(&database.User{
				Name:  userName,
				Email: userName + s.DefaultEmailDomain,
			}), ShouldResemble, *thingPost)

			// Get s3 post and verify deletion time automated with creation date.
			filter = &database.GetThingsParams{
				FilterOnType:   database.ThingsTypeS3,
				OrderBy:        database.OrderByType,
				OrderDirection: database.OrderAsc,
				Page:           1,
				ThingsPerPage:  1,
			}

			things, err = client.GetThings(filter)
			So(err, ShouldBeNil)
			So(len(things), ShouldEqual, 1)
			So(things[0].Remove, ShouldEqual, things[0].CreationDate.ValueOrZero().Add(fiveYear))

			things[0].Remove = time.Time{}
			things[0].CreationDate = null.Time{}
			thingTwoPost.CreationDate = null.Time{}

			So(things[0].ToCreateParams(&database.User{
				Name:  userName,
				Email: userName + s.DefaultEmailDomain,
			}), ShouldResemble, *thingTwoPost)

			// Getting both with no Type filter.
			filter = &database.GetThingsParams{
				FilterOnType:   database.ThingsTypeNil,
				OrderBy:        database.OrderByType,
				OrderDirection: database.OrderAsc,
				Page:           1,
				ThingsPerPage:  2,
			}

			things, err = client.GetThings(filter)
			So(err, ShouldBeNil)
			So(len(things), ShouldEqual, 2)

			things[0].Remove = time.Time{}
			things[1].Remove = time.Time{}
			things[1].CreationDate = null.Time{}
			thingTwoPost.CreationDate = null.Time{}

			So(things[0].ToCreateParams(&database.User{
				Name:  userName,
				Email: userName + s.DefaultEmailDomain,
			}), ShouldResemble, *thingPost)
			So(things[1].ToCreateParams(&database.User{
				Name:  userName,
				Email: userName + s.DefaultEmailDomain,
			}), ShouldResemble, *thingTwoPost)

			// Get with just ID
			filter = &database.GetThingsParams{
				FilterOnType:   database.ThingsTypeNil,
				OrderBy:        database.OrderByType,
				OrderDirection: database.OrderAsc,
				Page:           1,
				ThingsPerPage:  2,
			}

			things, err = client.GetThings(filter)
			So(err, ShouldBeNil)
			So(len(things), ShouldEqual, 2)

			thing, err := client.GetThingByID(things[0].ID)
			So(err, ShouldBeNil)
			So(thing, ShouldResemble, &things[0])

			Convey("You can Subscribe to things and get all of your subscriptions", func() {
				userPost := &database.User{
					Name:  "exampleSub",
					Email: "exampleSub" + s.DefaultEmailDomain,
				}
				err := client.CreateUser(userPost)
				So(err, ShouldBeNil)
				So(len(mdb.users), ShouldEqual, 2)

				err = client.Subscribe(&database.NewSubscriptionParams{Username: userPost.Name, ThingID: things[0].ID})
				So(err, ShouldBeNil)
				err = client.Subscribe(&database.NewSubscriptionParams{Username: userPost.Name, ThingID: things[1].ID})
				So(err, ShouldBeNil)

				subscribedThings, err := client.GetSubscriptions(userPost.Name)
				So(err, ShouldBeNil)
				So(subscribedThings[0].ID, ShouldEqual, things[0].ID)
				So(len(subscribedThings), ShouldEqual, 2)

				Convey("You can get information about subscribers to the thing created", func() {
					// Test get a subscriber with thingID.
					u, err := client.GetUserByName(userName)
					So(err, ShouldBeNil)

					subs, err := client.GetSubscribers(things[0].ID)
					So(err, ShouldBeNil)
					So(len(subs), ShouldEqual, 2)
					So(subs[0].UserID, ShouldResemble, u.ID)

					// Test get a thing and its subscriber details.
					thingAndSubs, err := client.GetThingAndSubscriberDetails(things[0].ID)
					So(err, ShouldBeNil)
					So(len(thingAndSubs.Subscribers), ShouldEqual, 2)

					So(thingAndSubs.Thing, ShouldResemble, things[0])
					So(thingAndSubs.Subscribers[0].ID, ShouldResemble, subs[0].UserID)
					So(thingAndSubs.Subscribers[1].Name, ShouldEqual, userPost.Name)
				})

				Convey("You can unsubscribe to a thing", func() {
					// TODO:
				})
			})
		})
	})
}
