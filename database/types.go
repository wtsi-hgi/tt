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

package database

import (
	"time"

	null "github.com/guregu/null/v5"
)

type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrBadType           = Error("Invalid things type")
	ErrBadOrderBy        = Error("Invalid order")
	ErrBadOrderDirection = Error("Invalid direction")
)

type ThingsType string

const (
	ThingsTypeNil       ThingsType = ""
	ThingsTypeDir       ThingsType = "dir"
	ThingsTypeFile      ThingsType = "file"
	ThingsTypeIrods     ThingsType = "irods"
	ThingsTypeOpenstack ThingsType = "openstack"
	ThingsTypeS3        ThingsType = "s3"
	ThingsTypeResource  ThingsType = "resource"
)

func ThingsTypes() []ThingsType {
	return []ThingsType{
		ThingsTypeDir,
		ThingsTypeFile,
		ThingsTypeIrods,
		ThingsTypeOpenstack,
		ThingsTypeS3,
		ThingsTypeResource,
	}
}

// NewThingsType converts the given str to a ThingsType, but only if it matches
// one of the allowed ThingsType* constants. Returns an error if not.
func NewThingsType(str string) (ThingsType, error) {
	var thingsType ThingsType

	switch ThingsType(str) {
	case ThingsTypeNil:
		thingsType = ThingsTypeNil
	case ThingsTypeDir:
		thingsType = ThingsTypeDir
	case ThingsTypeFile:
		thingsType = ThingsTypeFile
	case ThingsTypeIrods:
		thingsType = ThingsTypeIrods
	case ThingsTypeOpenstack:
		thingsType = ThingsTypeOpenstack
	case ThingsTypeS3:
		thingsType = ThingsTypeS3
	case ThingsTypeResource:
		thingsType = ThingsTypeResource
	default:
		return "", ErrBadType
	}

	return thingsType, nil
}

type OrderBy string

const (
	OrderByAddress OrderBy = "address"
	OrderByType    OrderBy = "type"
	OrderByReason  OrderBy = "reason"
	OrderByRemove  OrderBy = "remove"
)

// NewOrderBy converts the given str to an OrderBy, but only if it matches
// one of the allowed OrderBy* constants. Returns an error if not. Blank str
// returns the default OrderByRemove.
func NewOrderBy(str string) (OrderBy, error) {
	var orderBy OrderBy

	switch OrderBy(str) {
	case OrderByAddress:
		orderBy = OrderByAddress
	case OrderByType:
		orderBy = OrderByType
	case OrderByReason:
		orderBy = OrderByReason
	case "", OrderByRemove:
		orderBy = OrderByRemove
	default:
		return "", ErrBadOrderBy
	}

	return orderBy, nil
}

type OrderDirection string

const (
	OrderAsc  OrderDirection = "ASC"
	OrderDesc OrderDirection = "DESC"
)

// NewOrderDirection converts the given str to an OrderDirection, but only if it
// matches one of the allowed OrderDirection* constants. Returns an error if
// not. Blank str returns the default OrderAsc.
func NewOrderDirection(str string) (OrderDirection, error) {
	var orderDir OrderDirection

	switch OrderDirection(str) {
	case "", OrderAsc:
		orderDir = OrderAsc
	case OrderDesc:
		orderDir = OrderDesc
	default:
		return "", ErrBadOrderDirection
	}

	return orderDir, nil
}

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

// GetThingsResult is the type returned by GetThings(). The Things property will
// contain the retrieved results. The LastPage property will tell you the last
// value of Page in your GetThingsParams that would return any Things given the
// same GetThingsParams.ThingsPerPage. If Page or GetThingsParams is 0, LastPage
// will always be 0.
type GetThingsResult struct {
	Things   []Thing
	LastPage int
}

type User struct {
	ID    uint32
	Name  string
	Email string
}

type CreateThingParams struct {
	Address        string
	Type           ThingsType
	Description    string
	Reason         string
	Remove         time.Time `time_format:"2006-01-02"`
	License        string
	Version        string
	Name           string
	URL            string
	DownloadMethod string
	RequestSource  string
	CreationDate   time.Time
	Creator        string // Creator must correspond to the Name of a User.
}

type Thing struct {
	// ID is the unique (auto-increment) id of the temporary thing.
	ID uint32
	// Address is the location of the resource, such as absolute file path on
	// disk or URL.
	Address string
	// Type is the type of thing.
	Type ThingsType
	// Created is the time that the thing was added to the database.
	Created time.Time
	// Description is a description of the thing.
	Description string
	// Reason is the reason why this is a temporary thing.
	Reason string
	// Remove is the date that the removal will take place. Even "immortal"
	// resources get a date to ensure we are reviewing usage; it might just be
	// set many years from creation.
	Remove time.Time
	// Warned1 is the date that subscribers were successfully sent their first
	// warning.
	Warned1 null.Time
	// Warned2 is the date that subscribers were successfully sent their second
	// warning.
	Warned2 null.Time
	// Removed is whether or not removal has taken place.
	Removed bool
	// License is the licensing of the thing, such as MIT, or details of more
	// restrictive ones.
	License string
	// Version is the version of the thing.
	Version string
	// Name is a friendly short name for the thing.
	Name string
	// URL is specific to resources retrieved from a URL, such as downloaded
	// resources.
	URL string
	// DownloadMethod is the way in which the thing was downloaded for users to
	// use.
	DownloadMethod string
	// RequestSource is where the request for the thing was made, such as a JIRA
	// ticket.
	RequestSource string
	// CreationDate is the date the real thing was created (as opposed to
	// Created, which is when the thing was added to the database).
	CreationDate time.Time
}

// ToCreateParams converts thing Thing to a CreateThingParams with given user
// name.
func (t Thing) ToCreateParams(u User) CreateThingParams {
	return CreateThingParams{
		Address:        t.Address,
		Type:           t.Type,
		Description:    t.Description,
		Reason:         t.Reason,
		Remove:         t.Remove,
		Version:        t.Version,
		License:        t.License,
		Name:           t.Name,
		URL:            t.URL,
		DownloadMethod: t.DownloadMethod,
		RequestSource:  t.RequestSource,
		CreationDate:   t.CreationDate,
		Creator:        u.Name,
	}
}

type Subscriber struct {
	UserID  uint32
	ThingID uint32
	Creator bool
}
