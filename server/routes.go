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

const perPage = 50

func (s *Server) pageRoot(c *gin.Context) {
	c.HTML(http.StatusOK, "templates/root.html", nil)
}

func (s *Server) getThings(c *gin.Context) {
	dir := c.Query("dir")
	sortCol := c.DefaultQuery("sort", "remove")

	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	var thingType database.ThingsType

	switch c.Query("type") {
	case string(database.ThingsTypeDir):
		thingType = database.ThingsTypeDir
	}

	result, err := s.db.GetThings(database.GetThingsParams{
		FilterOnType:   thingType,
		OrderBy:        database.OrderBy(sortCol),
		OrderDirection: database.OrderDirection(dir),
		Page:           page,
		ThingsPerPage:  perPage,
	})
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)

		return
	}

	c.HTML(http.StatusOK, "templates/things.html", result.Things)
}

func (s *Server) postThing(c *gin.Context) {
	var postedThing database.CreateThingParams

	if err := c.ShouldBind(&postedThing); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)

		return
	}

	_, err := database.NewThingsType(string(postedThing.Type))
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)

		return
	}

	thing, err := s.db.CreateThing(postedThing)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)

		return
	}

	err = s.broadcastNewThing(thing)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)

		return
	}

	c.Status(http.StatusOK)
}

func (s *Server) deleteThing(c *gin.Context) {
	thingID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)

		return
	}

	err = s.db.DeleteThing(uint32(thingID))
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)

		return
	}

	c.Status(http.StatusOK)
}
