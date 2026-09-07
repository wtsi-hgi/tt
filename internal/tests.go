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

package internal

import (
	"fmt"
	"time"

	"github.com/guregu/null/v5"
	"github.com/wtsi-hgi/tt/database"
)

// GetExampleData returns 2 users and a selection of Things that they created,
// 2 things per ThingsType, and with property values that would all sort
// differently to each other.
func GetExampleData() ([]database.User, []database.Thing, []database.Subscriber) {
	user1 := exampleUser(1)
	user2 := exampleUser(2)

	i := uint32(0)
	year := uint32(1970)
	thingsTypes := []database.ThingsType{
		database.ThingsTypeIrods,
		database.ThingsTypeDir,
		database.ThingsTypeS3,
		database.ThingsTypeFile,
		database.ThingsTypeOpenstack,
	}
	thingsPerType := 2
	numThings := len(thingsTypes) * thingsPerType
	expectedThings := make([]database.Thing, numThings)
	expectedSubs := make([]database.Subscriber, numThings)
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

			remove := dateFromYear(year + i)

			expectedThing := database.Thing{
				ID:          i + 1,
				Address:     addresses[i],
				Type:        thingType,
				Description: "desc",
				Reason:      reasons[i],
				Remove:      remove,
			}

			expectedThings[i] = expectedThing

			expectedSubs[i] = database.Subscriber{
				UserID:  creator.ID,
				ThingID: expectedThing.ID,
				Creator: true,
			}

			i++
		}
	}

	return []database.User{user1, user2}, expectedThings, expectedSubs
}

func dateFromYear(year uint32) time.Time {
	remove, _ := time.Parse(time.DateOnly, fmt.Sprintf("%d-01-02", year))

	return remove
}

func exampleUser(id uint32) database.User {
	return database.User{
		ID:    id,
		Name:  fmt.Sprintf("user%d", id),
		Email: fmt.Sprintf("user%d@example.com", id),
	}
}

func GetExampleResourceData() (database.User, database.Thing, database.Subscriber) {
	expectedThing := database.Thing{
		ID:             1,
		Address:        "address",
		Type:           database.ThingsTypeResource,
		Description:    "desc",
		Reason:         "reason",
		Remove:         dateFromYear(uint32(1971)),
		Version:        "2",
		License:        "MIT",
		Name:           "ResourceName",
		URL:            "example.com",
		DownloadMethod: "command line",
		RequestSource:  "Jira",
		CreationDate:   null.TimeFrom(dateFromYear(uint32(1970))),
	}

	creator := exampleUser(1)

	expectedSub := database.Subscriber{
		UserID:  creator.ID,
		ThingID: expectedThing.ID,
		Creator: true,
	}

	return creator, expectedThing, expectedSub
}
