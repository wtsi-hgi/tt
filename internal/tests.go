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

	"github.com/wtsi-hgi/tt/database"
)

// GetExampleData returns 2 users and a selection of Things that they created,
// 2 things per ThingsType, and with property values that would all sort
// differently to each other.
func GetExampleData() ([]database.User, []database.Thing, []database.Subscriber) {
	emailSuffix := "@example.com"
	u1 := "user1"
	u2 := "user2"

	user1 := database.User{
		ID:    1,
		Name:  u1,
		Email: u1 + emailSuffix,
	}

	user2 := database.User{
		ID:    2,
		Name:  u2,
		Email: u2 + emailSuffix,
	}

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

			remove, _ := time.Parse(time.DateOnly, fmt.Sprintf("%d-01-02", year+i))

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
