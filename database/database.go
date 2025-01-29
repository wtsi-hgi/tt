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

// Queries are used to interact with a database of Things, Users and
// Subscribers.
type Queries interface {
	// CreateUser creates a new user with the given name and email. The returned
	// user will have its ID set.
	CreateUser(name, email string) (*User, error)

	// CreateThing creates a new Thing with the given details. The returned
	// Thing will have its ID set to an auto-increment value, and Created time
	// set to now. The supplied Creator must match the Name of an existing User,
	// and will be recored as a Subscriber of the new Thing.
	CreateThing(args CreateThingParams) (*Thing, error)

	// GetThings returns things that match the given parameters. Also in the
	// result is the last page that would return things if Page and
	// ThingsPerPage are > 0.
	GetThings(params GetThingsParams) (*GetThingsResult, error)

	// DeleteUser deletes the user with the given ID. This will also delete any
	// subscriptions the user had (but not any Things the user created).
	DeleteUser(id uint32) error

	// DeleteThing deletes the thing with the given ID.
	DeleteThing(id uint32) error

	// Close releases any resources associated with doing the Queries, such as
	// closing database handles.
	Close() error
}
