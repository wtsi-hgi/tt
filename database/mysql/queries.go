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
	"errors"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/wtsi-hgi/tt/database"
)

const ErrNoUser = database.Error("No User found with that name")

const createUser = `INSERT INTO users (name, email) VALUES (?, ?)`

// CreateUser creates a new user with the given name and email. The returned
// user will have its ID set.
func (m *DB) CreateUser(name, email string) (*database.User, error) {
	id, err := createRow(m.pool, createUser, name, email)
	if err != nil {
		return nil, err
	}

	return &database.User{
		ID:    id,
		Name:  name,
		Email: email,
	}, nil
}

type executor interface {
	Exec(query string, args ...any) (sql.Result, error)
}

func createRow(db executor, sql string, args ...any) (uint32, error) {
	result, err := db.Exec(sql, args...)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return uint32(id), nil //nolint: gosec // sql is a 32bit int
}

const getUserByName = `
SELECT id, name, email
FROM users
WHERE name = ?
`

// GetUserByName returns the user with the given name.
func (m *DB) GetUserByName(name string) (*database.User, error) {
	rows, err := m.pool.Query(getUserByName, name)
	if err != nil {
		return nil, err
	}

	return scanRowForUser(rows)
}

const getUserByID = `
SELECT id, name, email
FROM users
WHERE id = ?
`

// GetUserByID returns the user with the given user ID.
func (m *DB) GetUserByID(id uint32) (*database.User, error) {
	rows, err := m.pool.Query(getUserByID, id)
	if err != nil {
		return nil, err
	}

	return scanRowForUser(rows)
}

func scanRowForUser(rows *sql.Rows) (*database.User, error) {
	defer rows.Close()

	if gotRow := rows.Next(); !gotRow {
		return nil, ErrNoUser
	}

	var user database.User

	if err := rows.Scan(
		&user.ID,
		&user.Name,
		&user.Email,
	); err != nil {
		return nil, err
	}

	if err := rows.Close(); err != nil {
		return nil, err
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &user, nil
}

const getSubscriberByThing = `
SELECT user_id, thing_id
FROM subscribers
WHERE thing_id = ?
`

// Get subscribers returns a list of subscribers to a given thing, given that things ID.
func (m *DB) GetSubscribers(thingID uint32) ([]database.Subscriber, error) {
	rows, err := m.pool.Query(getSubscriberByThing, thingID)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var subscribers []database.Subscriber

	for rows.Next() {
		var subscriber database.Subscriber

		if err = rows.Scan(
			&subscriber.UserID,
			&subscriber.ThingID,
		); err != nil {
			return nil, err
		}

		subscribers = append(subscribers, subscriber)
	}

	if err = rows.Close(); err != nil {
		return nil, err
	}

	return subscribers, rows.Err()
}

const createThing = `
INSERT INTO things (
  creator_id, address, type, created, description, reason, remove, license, ` +
	`version, name, url, download_method, request_source, creation_date
) VALUES (
  ?,?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
)
`

const createSubscription = `
INSERT INTO subscribers (
  user_id, thing_id
) VALUES (
  ?, ?
)
`

// CreateThing creates a new thing with the given details. The returned thing
// will have its ID set to an auto-increment value, and Created time set to now.
// The supplied Creator must match the Name of an existing User, and will be
// recored as a Subscriber of the new Thing.
func (m *DB) CreateThing(args database.CreateThingParams) (*database.Thing, error) { //nolint:funlen
	created := time.Now()

	tx, err := m.pool.Begin()
	if err != nil {
		return nil, err
	}

	id, err := createRow(tx, createThing,
		args.CreatorID,
		args.Address,
		args.Type,
		created,
		args.Description,
		args.Reason,
		args.Remove,
		args.License,
		args.Version,
		args.Name,
		args.URL,
		args.DownloadMethod,
		args.RequestSource,
		args.CreationDate,
	)
	if err != nil {
		errRollback := tx.Rollback()

		return nil, errors.Join(err, errRollback)
	}

	_, err = tx.Exec(createSubscription, args.CreatorID, id)
	if err != nil {
		errRollback := tx.Rollback()

		return nil, errors.Join(err, errRollback)
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return &database.Thing{
		ID:             id,
		CreatorID:      args.CreatorID,
		Address:        args.Address,
		Type:           args.Type,
		Created:        created,
		Description:    args.Description,
		Reason:         args.Reason,
		Remove:         args.Remove,
		License:        args.License,
		Version:        args.Version,
		Name:           args.Name,
		URL:            args.URL,
		DownloadMethod: args.DownloadMethod,
		RequestSource:  args.RequestSource,
		CreationDate:   args.CreationDate,
	}, nil
}

const getThings = `
SELECT things.id,creator_id, address, type, created, description, reason, remove, warned1, warned2, ` +
	`removed, license, version, name, url, download_method, request_source, creation_date
FROM things
`

//!! CASE WHEN users.id IS NOT NULL THEN TRUE
// 		ELSE FALSE END AS subscribed
//!! LEFT JOIN `subscriptions` ON (things.id = subscriptions.thing_id)
// LEFT JOIN `users` ON (subscriptions.user_id = users.id AND users.name = ?)

// GetThings returns things that match the given parameters. Also in the result
// is the last page that would return things if Page and ThingsPerPage are > 0.
func (m *DB) GetThings(params database.GetThingsParams) (*database.GetThingsResult, error) { //nolint:funlen
	rows, err := m.runGetThingsQuery(params)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var things []database.Thing

	for rows.Next() {
		thing, errc := getThingRowScan(rows)
		if errc != nil {
			return nil, errc
		}

		things = append(things, *thing)
	}

	if err = rows.Close(); err != nil {
		return nil, err
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	lastPage, errc := m.calculateLastPage(params)
	if errc != nil {
		return nil, errc
	}

	return &database.GetThingsResult{
		Things:   things,
		LastPage: lastPage,
	}, nil
}

const getThingByID = `
SELECT things.id,creator_id, address, type, created, description, reason, remove, warned1, warned2, ` +
	`removed, license, version, name, url, download_method, request_source, creation_date
FROM things
WHERE things.id = ?
`

// GetThingByID returns a Thing with the given Thing ID.
func (m *DB) GetThingByID(id uint32) (*database.Thing, error) {
	rows, err := m.pool.Query(getThingByID, id)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	if gotRow := rows.Next(); !gotRow {
		return nil, ErrNoUser
	}

	thing, err := getThingRowScan(rows)
	if err != nil {
		return nil, err
	}

	if err := rows.Close(); err != nil {
		return nil, err
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return thing, nil
}

// const getThingIDByUserSubscribed = `
// SELECT thing_id
// FROM subscribers
// WHERE user_id = ?
// `
// // GetSubscriptions returns a list of Things that a given user is subscribed to.
// func (m *DB) GetSubscriptions(userID uint32) ([]database.Thing, error) {
// 	rows, err := m.pool.Query(getThingIDByUserSubscribed, userID)
// 	if err != nil {
// 		return nil, err
// 	}

// 	defer rows.Close()

// 	var things []database.Thing

// 	for rows.Next() {
// 		var thingID uint32

// 		if err = rows.Scan(
// 			&thingID,
// 		); err != nil {
// 			return nil, err
// 		}

// 		thing, errc := m.GetThingByID(thingID)
// 		if errc != nil {
// 			return nil, errc
// 		}

// 		things = append(things, *thing)
// 	}

// 	if err = rows.Close(); err != nil {
// 		return nil, err
// 	}

// 	if err := rows.Err(); err != nil {
// 		return nil, err
// 	}

// 	return things, nil
// }

// // When using ID:
// const getThingIDByUserSubscribed = `
// SELECT thing.id, thing.name, creator_id, address, type, created, description, reason, remove, warned1, warned2, ` +
// 	`removed, license, version, url, download_method, request_source, creation_date
// FROM subscribers LEFT JOIN things ON(thing_id = things.id)
// WHERE user_id = ?
// `

// select the uid from user > select the thing id from sub > select the thing from thing:
const getThingsByUsernameSubscribed = `
SELECT things.id, things.creator_id, things.address, things.type, things.created,
       things.description, things.reason, things.remove, things.warned1, things.warned2,
       things.removed, things.license, things.version, things.name, things.url,
       things.download_method, things.request_source, things.creation_date
FROM users
JOIN subscribers ON subscribers.user_id = users.id
JOIN things ON things.id = subscribers.thing_id
WHERE users.name = ?
`

// GetSubscriptions returns a list of Things that a given user is subscribed to.
func (m *DB) GetSubscriptions(username string) ([]database.Thing, error) {
	rows, err := m.pool.Query(getThingsByUsernameSubscribed, username)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var things []database.Thing

	for rows.Next() {
		thing, errc := getThingRowScan(rows)
		if errc != nil {
			return nil, errc
		}

		things = append(things, *thing)
	}

	if err = rows.Close(); err != nil {
		return nil, err
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return things, nil
}

const subscribe = `INSERT INTO subscribers (user_id, thing_id) VALUES (?, ?);` //nolint:unused

// PostSubscription records that a user is subscribed to a thing.
func (m *DB) PostSubscription(username string, thingID uint32) error {
	// TODO simplify to one query?
	user, err := m.GetUserByName(username)
	if err != nil {
		return err
	}

	_, err = createRow(m.pool, subscribe, user.ID, thingID)
	if err != nil {
		return err
	}

	return nil
}

const unsubscribe = `DELETE FROM subscribers WHERE user_id = ? AND thing_id = ?;` //nolint:unused
// func (m *DB) DeleteSubscription(username string, thingID uint32) error {
// 	user, err := m.GetUserByName(username)
// 	if err != nil {
// 		return err
// 	}

// 	_, err = m.pool.Exec(unsubscribe, user.ID, thingID)
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

func getThingRowScan(rows *sql.Rows) (*database.Thing, error) {
	var thing database.Thing

	if err := rows.Scan(
		&thing.ID,
		&thing.CreatorID,
		&thing.Address,
		&thing.Type,
		&thing.Created,
		&thing.Description,
		&thing.Reason,
		&thing.Remove,
		&thing.Warned1,
		&thing.Warned2,
		&thing.Removed,
		&thing.License,
		&thing.Version,
		&thing.Name,
		&thing.URL,
		&thing.DownloadMethod,
		&thing.RequestSource,
		&thing.CreationDate,
	); err != nil {
		return nil, err
	}

	return &thing, nil
}

func (m *DB) runGetThingsQuery(params database.GetThingsParams) (*sql.Rows, error) {
	var sql strings.Builder

	sql.WriteString(getThings)
	getThingsParamsToSQL(params, &sql)

	rows, err := m.pool.Query(sql.String())
	if err != nil {
		return nil, err
	}

	return rows, nil
}

func getThingsParamsToSQL(params database.GetThingsParams, sql *strings.Builder) {
	whereSQL(params, sql)
	orderSQL(params, sql)
	limitSQL(params, sql)
}

func whereSQL(params database.GetThingsParams, sql *strings.Builder) {
	if params.FilterOnType == database.ThingsTypeNil {
		return
	}

	sql.WriteString("\nWHERE type = '")
	sql.WriteString(string(params.FilterOnType))
	sql.WriteString("'")
}

func orderSQL(params database.GetThingsParams, sql *strings.Builder) {
	sql.WriteString("\nORDER BY ")

	if params.OrderBy == "" {
		sql.WriteString(string(database.OrderByRemove))
	} else {
		sql.WriteString(string(params.OrderBy))
	}

	sql.WriteString(" ")

	if params.OrderDirection == "" {
		sql.WriteString(string(database.OrderAsc))
	} else {
		sql.WriteString(string(params.OrderDirection))
	}
}

func limitSQL(params database.GetThingsParams, sql *strings.Builder) {
	if params.Page < 1 || params.ThingsPerPage < 1 {
		return
	}

	limit := strconv.Itoa(params.ThingsPerPage)
	offset := strconv.Itoa((params.Page - 1) * params.ThingsPerPage)

	sql.WriteString("\nLIMIT ")
	sql.WriteString(limit)
	sql.WriteString(" OFFSET ")
	sql.WriteString(offset)
}

const countThings = `SELECT COUNT(*) FROM things`

func (m *DB) calculateLastPage(params database.GetThingsParams) (int, error) {
	if params.Page < 1 || params.ThingsPerPage < 1 {
		return 0, nil
	}

	var (
		sql   strings.Builder
		count int64
	)

	sql.WriteString(countThings)
	whereSQL(params, &sql)

	row := m.pool.QueryRow(sql.String())

	if err := row.Scan(&count); err != nil {
		return 0, err
	}

	return int(math.Ceil(float64(count) / float64(params.ThingsPerPage))), nil
}

const deleteUser = `DELETE FROM users WHERE id = ?`

// DeleteUser deletes the user with the given ID. This will also delete any
// subscriptions the user had (but not any Things the user created).
func (m *DB) DeleteUser(id uint32) error {
	_, err := m.pool.Exec(deleteUser, id)

	return err
}

const deleteThing = `DELETE FROM things WHERE id = ?`

// DeleteThing deletes the thing with the given ID.
func (m *DB) DeleteThing(id uint32) error {
	_, err := m.pool.Exec(deleteThing, id)

	return err
}

const extendRemoval = `UPDATE things SET remove = ?, warned1 = NULL, warned2 = NULL WHERE id = ?;` //nolint:unused

const updateDescription = `UPDATE things SET description = ? WHERE id = ?;` //nolint:unused

const firstWarningSent = `UPDATE things SET warned1 = ? WHERE id = ?;` //nolint:unused

const secondWarningSent = `UPDATE things SET warned2 = ? WHERE id = ?;` //nolint:unused

const listSubscribers = `SELECT user_id, thing_id FROM subscribers WHERE thing_id = ?;` //nolint:unused

const listSubscriptions = `SELECT user_id, thing_id FROM subscribers WHERE user_id = ?;` //nolint:unused
