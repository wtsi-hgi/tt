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

type Subscriber struct {
	UserID  uint32
	ThingID uint32
}

type Thing struct {
	ID          uint32
	Address     string
	Type        ThingsType
	Created     time.Time
	Description string
	Reason      string
	Remove      time.Time
	Warned1     null.Time
	Warned2     null.Time
	Removed     bool
}

type User struct {
	ID    uint32
	Name  string
	Email string
}
