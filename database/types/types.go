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

package types

import (
	"time"

	null "github.com/guregu/null/v5"
)

type ThingsType string

const (
	ThingsTypeFile      ThingsType = "file"
	ThingsTypeDir       ThingsType = "dir"
	ThingsTypeIrods     ThingsType = "irods"
	ThingsTypeS3        ThingsType = "s3"
	ThingsTypeOpenstack ThingsType = "openstack"
)

func ThingsTypes() []ThingsType {
	return []ThingsType{
		ThingsTypeFile,
		ThingsTypeDir,
		ThingsTypeIrods,
		ThingsTypeS3,
		ThingsTypeOpenstack,
	}
}

type OrderBy string

const (
	OrderByAddres OrderBy = "address"
	OrderByType   OrderBy = "type"
	OrderByReason OrderBy = "reason"
	OrderByRemove OrderBy = "remove"
)

type OrderDirection string

const (
	OrderAsc  OrderDirection = "ASC"
	OrderDesc OrderDirection = "DESC"
)

// GetThingsParams, when default value and provided to GetThings(), will get
// all things. Optionally set any of the values to filter, order or get a
// certain page of results.
type GetThingsParams struct {
	FilterOnType   ThingsType
	OrderBy        OrderBy        // defaults to OrderByRemove
	OrderDirection OrderDirection // defaults to OrderAsc
	Page           int            // treated as 0 if ThingsPerPage is < 1
	ThingsPerPage  int            // treated as infinite if Page is < 1
}

type User struct {
	ID    uint32
	Name  string
	Email string
}

type CreateThingParams struct {
	Address     string
	Type        ThingsType
	Creator     *User
	Description string
	Reason      string
	Remove      time.Time
}

type Thing struct {
	ID          uint32
	Address     string
	Type        ThingsType
	Creator     *User
	Created     time.Time
	Description string
	Reason      string
	Remove      time.Time
	Warned1     null.Time
	Warned2     null.Time
	Removed     bool
}

type Subscriber struct {
	UserID  uint32
	ThingID uint32
}
