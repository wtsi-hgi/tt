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
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/wtsi-hgi/tt/database/types"
)

const createUser = `INSERT INTO users (name, email) VALUES (?, ?)`

// CreateUser creates a new user with the given name and email. The returned
// user will have its ID set.
func (m *MySQLDB) CreateUser(name, email string) (*types.User, error) {
	id, err := m.createRow(createUser, name, email)
	if err != nil {
		return nil, err
	}

	return &types.User{
		ID:    id,
		Name:  name,
		Email: email,
	}, nil
}

func (m *MySQLDB) createRow(sql string, args ...any) (uint32, error) {
	result, err := m.pool.Exec(sql, args...)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return uint32(id), nil
}

const createThing = `
INSERT INTO things (
  address, type, creator, created, description, reason, remove
) VALUES (
  ?, ?, ?, ?, ?, ?, ?
)
`

// CreateThing creates a new thing with the given details. The returned thing
// will have its ID set to an auto-increment value, and Created time set to now.
func (m *MySQLDB) CreateThing(args types.CreateThingParams) (*types.Thing, error) {
	created := time.Now()

	id, err := m.createRow(createThing,
		args.Address,
		args.Type,
		args.Creator.ID,
		created,
		args.Description,
		args.Reason,
		args.Remove,
	)
	if err != nil {
		return nil, err
	}

	return &types.Thing{
		ID:          uint32(id),
		Address:     args.Address,
		Type:        args.Type,
		Creator:     args.Creator,
		Created:     created,
		Description: args.Description,
		Reason:      args.Reason,
		Remove:      args.Remove,
	}, nil
}

const getThings = `
SELECT things.id, address, type, users.id, users.name, users.email, created, description, reason, remove, warned1, warned2, removed
FROM things
INNER JOIN users ON things.creator=users.id
`

// GetThings returns things that match the given parameters. Also in the result
// is the last page that would return things if Page and ThingsPerPage are > 0.
func (m *MySQLDB) GetThings(params types.GetThingsParams) (*types.GetThingsResult, error) {
	var sql strings.Builder

	sql.WriteString(getThings)
	getThingsParamsToSQL(params, &sql)

	rows, err := m.pool.Query(sql.String())
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var things []types.Thing

	for rows.Next() {
		var thing types.Thing
		var user types.User

		if err := rows.Scan(
			&thing.ID,
			&thing.Address,
			&thing.Type,
			&user.ID,
			&user.Name,
			&user.Email,
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

		thing.Creator = &user

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

	return &types.GetThingsResult{
		Things:   things,
		LastPage: lastPage,
	}, nil
}

func getThingsParamsToSQL(params types.GetThingsParams, sql *strings.Builder) {
	whereSQL(params, sql)
	orderSQL(params, sql)
	limitSQL(params, sql)
}

func whereSQL(params types.GetThingsParams, sql *strings.Builder) {
	if params.FilterOnType == "" {
		return
	}

	sql.WriteString("\nWHERE type = '")
	sql.WriteString(string(params.FilterOnType))
	sql.WriteString("'")
}

func orderSQL(params types.GetThingsParams, sql *strings.Builder) {
	sql.WriteString("\nORDER BY ")

	if params.OrderBy == "" {
		sql.WriteString(string(types.OrderByRemove))
	} else {
		sql.WriteString(string(params.OrderBy))
	}

	sql.WriteString(" ")

	if params.OrderDirection == "" {
		sql.WriteString(string(types.OrderAsc))
	} else {
		sql.WriteString(string(params.OrderDirection))
	}
}

func limitSQL(params types.GetThingsParams, sql *strings.Builder) {
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

func (m *MySQLDB) calculateLastPage(params types.GetThingsParams) (int, error) {
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
