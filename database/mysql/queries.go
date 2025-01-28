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
func (m *MySQLDB) CreateUser(name, email string) (*database.User, error) {
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

	return uint32(id), nil
}

const getUserByName = `
SELECT id, name, email
FROM users
WHERE name = ?
`

// GetUserByName returns the user with the given name.
func (m *MySQLDB) GetUserByName(name string) (*database.User, error) {
	rows, err := m.pool.Query(getUserByName, name)
	if err != nil {
		return nil, err
	}

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

const createThing = `
INSERT INTO things (
  address, type, created, description, reason, remove
) VALUES (
  ?, ?, ?, ?, ?, ?
)
`

const createSubscription = `
INSERT INTO subscribers (
  user_id, thing_id, creator
) VALUES (
  ?, ?, ?
)
`

// CreateThing creates a new thing with the given details. The returned thing
// will have its ID set to an auto-increment value, and Created time set to now.
// The supplied Creator must match the Name of an existing User, and will be
// recored as a Subscriber of the new Thing.
func (m *MySQLDB) CreateThing(args database.CreateThingParams) (*database.Thing, error) {
	created := time.Now()

	user, err := m.GetUserByName(args.Creator)
	if err != nil {
		return nil, err
	}

	tx, err := m.pool.Begin()
	if err != nil {
		return nil, err
	}

	id, err := createRow(tx, createThing,
		args.Address,
		args.Type,
		created,
		args.Description,
		args.Reason,
		args.Remove,
	)
	if err != nil {
		tx.Rollback()

		return nil, err
	}

	_, err = tx.Exec(createSubscription, user.ID, id, 1)
	if err != nil {
		tx.Rollback()

		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return &database.Thing{
		ID:          uint32(id),
		Address:     args.Address,
		Type:        args.Type,
		Created:     created,
		Description: args.Description,
		Reason:      args.Reason,
		Remove:      args.Remove,
	}, nil
}

const getThings = `
SELECT things.id, address, type, created, description, reason, remove, warned1, warned2, removed
FROM things
`

// GetThings returns things that match the given parameters. Also in the result
// is the last page that would return things if Page and ThingsPerPage are > 0.
func (m *MySQLDB) GetThings(params database.GetThingsParams) (*database.GetThingsResult, error) {
	var sql strings.Builder

	sql.WriteString(getThings)
	getThingsParamsToSQL(params, &sql)

	rows, err := m.pool.Query(sql.String())
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var things []database.Thing

	for rows.Next() {
		var thing database.Thing

		if err := rows.Scan(
			&thing.ID,
			&thing.Address,
			&thing.Type,
			&thing.Created,
			&thing.Description,
			&thing.Reason,
			&thing.Remove,
			&thing.Warned1,
			&thing.Warned2,
			&thing.Removed,
		); err != nil {
			return nil, err
		}

		things = append(things, thing)
	}

	if err := rows.Close(); err != nil {
		return nil, err
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	lastPage, err := m.calculateLastPage(params)
	if err != nil {
		return nil, err
	}

	return &database.GetThingsResult{
		Things:   things,
		LastPage: lastPage,
	}, nil
}

func getThingsParamsToSQL(params database.GetThingsParams, sql *strings.Builder) {
	whereSQL(params, sql)
	orderSQL(params, sql)
	limitSQL(params, sql)
}

func whereSQL(params database.GetThingsParams, sql *strings.Builder) {
	if params.FilterOnType == "" {
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

func (m *MySQLDB) calculateLastPage(params database.GetThingsParams) (int, error) {
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
// things created by this user.
func (m *MySQLDB) DeleteUser(id uint32) error {
	_, err := m.pool.Exec(deleteUser, id)

	return err
}

const deleteThing = `DELETE FROM things WHERE id = ?`

// DeleteThing deletes the thing with the given ID.
func (m *MySQLDB) DeleteThing(id uint32) error {
	_, err := m.pool.Exec(deleteThing, id)

	return err
}

const extendRemoval = `
UPDATE things
SET remove = ?, warned1 = NULL, warned2 = NULL
WHERE id = ?
`

const updateDescription = `
UPDATE things
SET description = ?
WHERE id = ?
`

const firstWarningSent = `
UPDATE things
SET warned1 = ?
WHERE id = ?
`

const secondWarningSent = `
UPDATE things
SET warned2 = ?
WHERE id = ?
`

const subscribe = `
INSERT INTO subscribers (
  user_id, thing_id
) VALUES (
  ?, ?
)
`

const unsubscribe = `
DELETE FROM subscribers
WHERE user_id = ? AND thing_id = ?
`

const listSubscribers = `
SELECT user_id, thing_id FROM subscribers
WHERE thing_id = ?
`

const listSubscriptions = `
SELECT user_id, thing_id FROM subscribers
WHERE user_id = ?
`
