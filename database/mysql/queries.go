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
