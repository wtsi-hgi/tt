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
