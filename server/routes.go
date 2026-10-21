/*******************************************************************************
 * Copyright (c) 2026 Genome Research Ltd.
 *
 * Authors:
 *	- Sendu Bala <sb10@sanger.ac.uk>
 *  - Amber Faruque <af35@sanger.ac.uk>
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
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	gas "github.com/wtsi-hgi/go-authserver"
	"github.com/wtsi-hgi/tt/database"
)

const (
	fiveYear = time.Hour * 24 * 365 * 5
	jsonType = "application/json"
)

type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrInvalidEmail    = Error("Invalid email")
	ErrInvalidUserName = Error("Invalid username")
	ErrInvalidThingID  = Error("Invalid thing ID")
	ErrUnauthorized    = Error("Unauthorized") //nolint:misspell
	ErrNoUser          = database.Error("No User found with that name")
)

const (
	pageRootPath     = "/"
	thingsPath       = "/things"
	thingIDPath      = "/things/:id"
	userPath         = "/user"
	thingsListenPath = "/things/listen"
	subscribersPath  = "/subscribers"

	EndPointAuthRoot               = gas.EndPointAuth + pageRootPath
	EndPointAuthUser               = gas.EndPointAuth + userPath
	EndPointAuthThings             = gas.EndPointAuth + thingsPath
	EndPointAuthThingID            = gas.EndPointAuth + thingIDPath
	EndPointAuthSseThingsEventName = thingsListenPath
	EndPointAuthSubscribers        = gas.EndPointAuth + subscribersPath
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
	filter, err := database.NewGetThingsParams(c.Query("type"), c.Query("sort"), c.Query("dir"), c.Query("page"), c.Query("per_page")) //nolint:lll
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err) //nolint: errcheck

		return
	}

	result, err := s.db.GetThings(filter)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err) //nolint: errcheck

		return
	}

	if c.Request.Header.Get("Accept") == jsonType {
		c.JSON(http.StatusOK, result.Things)

		return
	}

	c.HTML(http.StatusOK, "templates/things.html", result.Things)
}

// getThingByID returns the row in the Things table in the database. The id of
// the desired thing can be supplied as a query param by the User.
func (s *Server) getThingByID(c *gin.Context) {
	tID := c.Param("id")
	if tID == "" {
		return
	}

	id, err := strconv.Atoi(tID)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err) //nolint: errcheck

		return
	}

	result, err := s.db.GetThingByID(uint32(id)) //nolint:gosec
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err) //nolint: errcheck

		return
	}

	if c.Request.Header.Get("Accept") == jsonType {
		c.JSON(http.StatusOK, result)

		return
	}

	c.HTML(http.StatusOK, "templates/things.html", result)
}

// postThing posts all required fields of a Thing to /things, along with Creator
// as the username of the person making this thing, and creates a new Thing
// and Subscriber in the database.
//
// Afterwards, it broadcasts the new Thing to all listeners of /things/listen
// using SSE.
func (s *Server) postThing(c *gin.Context) {
	var postedThing database.CreateThingParams

	err := s.postThingValidate(c, &postedThing)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err) //nolint: errcheck

		return
	}

	user := validateOrCreateUser(c, s, postedThing)
	if user == nil {
		return
	}

	postedThing.CreatorID = user.ID

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

	c.Status(http.StatusNoContent)
}

func validateOrCreateUser(c *gin.Context, s *Server, postedThing database.CreateThingParams) *database.User {
	if !s.AllowedAccess(c, postedThing.Creator) {
		c.AbortWithError(http.StatusUnauthorized, ErrUnauthorized) //nolint: errcheck

		return nil
	}

	user, err := s.db.GetUserByName(postedThing.Creator)
	if errors.Is(err, ErrNoUser) {
		user, err = s.db.CreateUser(postedThing.Creator, postedThing.Creator+s.DefaultEmailDomain)
	}

	if err != nil && !errors.Is(err, ErrNoUser) {
		c.AbortWithError(http.StatusInternalServerError, err) //nolint: errcheck

		return nil
	}

	return user
}

func (s *Server) postThingValidate(c *gin.Context, postedThing *database.CreateThingParams) error {
	if err := c.ShouldBind(postedThing); err != nil {
		return err
	}

	// Set defaults.
	if postedThing.Remove.IsZero() {
		setRemovalDate(postedThing)
	}

	// Validating values.
	err := database.ValidateCreateThingsParams(*postedThing)
	if err != nil {
		return err
	}

	return nil
}
func setRemovalDate(postedThing *database.CreateThingParams) {
	if postedThing.CreationDate.Valid {
		postedThing.Remove = postedThing.CreationDate.Time.Add(fiveYear)

		return
	}

	postedThing.Remove = time.Now().Add(fiveYear)
}

// deleteThing deletes the thing with the id in the url /things/id from the
// database.
func (s *Server) deleteThing(c *gin.Context) {
	thingID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err) //nolint: errcheck

		return
	}

	err = s.db.DeleteThing(uint32(thingID))
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err) //nolint: errcheck

		return
	}

	c.Status(http.StatusOK)
}

func (s *Server) postUser(c *gin.Context) {
	var userPost database.User

	if err := c.ShouldBind(&userPost); err != nil {
		c.AbortWithError(http.StatusBadRequest, err) //nolint: errcheck

		return
	}

	if !s.AllowedAccess(c, userPost.Name) {
		c.AbortWithError(http.StatusUnauthorized, ErrUnauthorized) //nolint: errcheck

		return
	}

	if userPost.Name == "" {
		c.AbortWithError(http.StatusBadRequest, ErrInvalidUserName) //nolint: errcheck

		return
	}

	if userPost.Email == "" {
		userPost.Email = userPost.Name + s.DefaultEmailDomain
	}

	_, err := s.db.CreateUser(userPost.Name, userPost.Email)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err) //nolint:errcheck

		return
	}

	c.Status(http.StatusNoContent)
}

func (s *Server) getUser(c *gin.Context) {
	username := c.Query("name")
	uID := c.Query("id")

	if username == "" && uID == "" {
		c.AbortWithError(http.StatusBadRequest, ErrInvalidUserName) //nolint: errcheck

		return
	}

	if username != "" {
		s.getUserByName(c, username)

		return
	}

	s.getUserByID(c, uID)
}

func (s *Server) getUserByName(c *gin.Context, username string) {
	result, err := s.db.GetUserByName(username)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err) //nolint: errcheck

		return
	}

	if c.Request.Header.Get("Accept") == jsonType {
		c.JSON(http.StatusOK, result)

		return
	}

	c.HTML(http.StatusOK, "templates/things.html", result)
}

func (s *Server) getUserByID(c *gin.Context, uID string) {
	id, err := strconv.Atoi(uID)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err) //nolint: errcheck

		return
	}

	result, err := s.db.GetUserByID(uint32(id)) //nolint:gosec
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err) //nolint: errcheck

		return
	}

	if c.Request.Header.Get("Accept") == jsonType {
		c.JSON(http.StatusOK, result)

		return
	}

	c.HTML(http.StatusOK, "templates/things.html", result)
}

func (s *Server) getSubscribers(c *gin.Context) {
	tID := c.Query("id")

	id, err := strconv.Atoi(tID)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, ErrInvalidThingID) //nolint: errcheck

		return
	}

	result, errc := s.db.GetSubscribers(uint32(id)) //nolint:gosec
	if errc != nil {
		c.AbortWithError(http.StatusBadRequest, errc) //nolint: errcheck

		return
	}

	if c.Request.Header.Get("Accept") == jsonType {
		c.JSON(http.StatusOK, result)

		return
	}

	c.HTML(http.StatusOK, "templates/things.html", result)
}
