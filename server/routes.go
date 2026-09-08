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

package server

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/wtsi-hgi/tt/database"
)

const (
	defaultPage    = 1
	defaultPerPage = 50
)

// pageRoot takes no user input; it's for the overall main html page at /.
func (s *Server) pageRoot(c *gin.Context) {
	c.HTML(http.StatusOK, "templates/root.html", nil)
}

// getThings returns table rows for each Thing in the database. User can supply
// url query values to /things? to alter which and how these are returned:
//
// sort=[address|type|reason|remove] : sort by chosen field in Thing, default
// remove
//
// dir=[ASC|DESC] : sort direction, default ascending
//
// type=[dir|file|irods|openstack|s3] : filter to only show this type of thing
//
// page=<int>&per_page=<int> : get a particular page of results, where each page
// has per_page Things. Page defaults to 1, and per_page defaults to 50.
func (s *Server) getThings(c *gin.Context) {
	orderBy, orderDirection, thingType, page, perPage, err := parseGetThingsParams(c)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err) //nolint: errcheck

		return
	}

	result, err := s.db.GetThings(database.GetThingsParams{
		FilterOnType:   thingType,
		OrderBy:        orderBy,
		OrderDirection: orderDirection,
		Page:           page,
		ThingsPerPage:  perPage,
	})
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err) //nolint: errcheck

		return
	}

	c.HTML(http.StatusOK, "templates/things.html", result.Things)
}

func parseGetThingsParams(c *gin.Context) (orderBy database.OrderBy,
	orderDirection database.OrderDirection, thingsType database.ThingsType, page int, perPage int, err error) {
	orderBy, err = database.NewOrderBy(c.Query("sort"))
	if err != nil {
		return //nolint:nakedret
	}

	orderDirection, err = database.NewOrderDirection(c.Query("dir"))
	if err != nil {
		return //nolint:nakedret
	}

	thingsType, err = database.NewThingsType(c.Query("type"))
	if err != nil {
		return //nolint:nakedret
	}

	page = atoiWithDefault(c.Query("page"), defaultPage)
	perPage = atoiWithDefault(c.Query("per_page"), defaultPerPage)

	return //nolint:nakedret
}

func atoiWithDefault(text string, defaultVal int) int {
	num, err := strconv.Atoi(text)

	if err != nil || num < 1 {
		return defaultVal
	}

	return num
}

// postThing posts all required fields of a Thing to /things, along with Creator
// as the username of the person making this thing, and creates a new Thing
// and Subscriber in the database.
//
// Afterwards, it broadcasts the new Thing to all listeners of /things/listen
// using SSE.
func (s *Server) postThing(c *gin.Context) {
	var postedThing database.CreateThingParams

	if err := c.ShouldBind(&postedThing); err != nil {
		c.AbortWithError(http.StatusBadRequest, err) //nolint: errcheck

		return
	}

	_, err := database.NewThingsType(string(postedThing.Type))
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err) //nolint: errcheck

		return
	}

	thing, err := s.db.CreateThing(postedThing)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err) //nolint: errcheck

		return
	}

	err = s.broadcastNewThing(thing)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err) //nolint: errcheck

		return
	}

	c.Status(http.StatusOK)
}

// deleteThing deletes the thing with the id in the url /things/id from the
// database.
func (s *Server) deleteThing(c *gin.Context) {
	thingID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err) //nolint: errcheck

		return
	}

	err = s.db.DeleteThing(uint32(thingID)) //nolint: gosec // sql is a 32bit int
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err) //nolint: errcheck

		return
	}

	c.Status(http.StatusOK)
}
