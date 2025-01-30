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
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestNewThingsTypes(t *testing.T) {
	Convey("You can convert strings to ThingsType*, unless it's invalid", t, func() {
		tt, err := NewThingsType("")
		So(err, ShouldBeNil)
		So(tt, ShouldEqual, ThingsTypeNil)

		tt, err = NewThingsType("dir")
		So(err, ShouldBeNil)
		So(tt, ShouldEqual, ThingsTypeDir)

		tt, err = NewThingsType("file")
		So(err, ShouldBeNil)
		So(tt, ShouldEqual, ThingsTypeFile)

		tt, err = NewThingsType("irods")
		So(err, ShouldBeNil)
		So(tt, ShouldEqual, ThingsTypeIrods)

		tt, err = NewThingsType("openstack")
		So(err, ShouldBeNil)
		So(tt, ShouldEqual, ThingsTypeOpenstack)

		tt, err = NewThingsType("s3")
		So(err, ShouldBeNil)
		So(tt, ShouldEqual, ThingsTypeS3)

		_, err = NewThingsType("invalid")
		So(err, ShouldNotBeNil)
	})
}

func TestNewOrderBy(t *testing.T) {
	Convey("You can convert strings to OrderBy*, unless it's invalid", t, func() {
		ob, err := NewOrderBy("")
		So(err, ShouldBeNil)
		So(ob, ShouldEqual, OrderByRemove)

		ob, err = NewOrderBy("address")
		So(err, ShouldBeNil)
		So(ob, ShouldEqual, OrderByAddres)

		ob, err = NewOrderBy("type")
		So(err, ShouldBeNil)
		So(ob, ShouldEqual, OrderByType)

		ob, err = NewOrderBy("reason")
		So(err, ShouldBeNil)
		So(ob, ShouldEqual, OrderByReason)

		ob, err = NewOrderBy("remove")
		So(err, ShouldBeNil)
		So(ob, ShouldEqual, OrderByRemove)

		_, err = NewOrderBy("invalid")
		So(err, ShouldNotBeNil)
	})
}

func TestNewOrderDirection(t *testing.T) {
	Convey("You can convert strings to OrderDirection*, unless it's invalid", t, func() {
		od, err := NewOrderDirection("")
		So(err, ShouldBeNil)
		So(od, ShouldEqual, OrderAsc)

		od, err = NewOrderDirection("ASC")
		So(err, ShouldBeNil)
		So(od, ShouldEqual, OrderAsc)

		od, err = NewOrderDirection("DESC")
		So(err, ShouldBeNil)
		So(od, ShouldEqual, OrderDesc)

		_, err = NewOrderDirection("invalid")
		So(err, ShouldNotBeNil)
	})
}
