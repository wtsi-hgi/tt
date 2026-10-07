/*******************************************************************************
* Copyright (c) 2026 Genome Research Ltd.
*
* Authors:
*	- Sendu Bala <sb10@sanger.ac.uk>
*   - Amber Faruque <af35@sanger.ac.uk>
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
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-resty/resty/v2"
	gas "github.com/wtsi-hgi/go-authserver"
	"github.com/wtsi-hgi/tt/database"
)

const (
	ErrInvalidInput = Error("Invalid input")
	ErrInternal     = Error("internal error")
)
const (
	serverTokenBasename = ".tt.token"
)

// Client is used to interact with the Server over the network, with
// authentication.
type Client struct {
	url  string
	cert string
	jwt  string
	// logger log15.Logger
}

type SubscriberWithUserDetails struct {
	database.User
	Creator bool
}

type ThingWithSubscriberDetails struct {
	database.Thing
	Subscribers []SubscriberWithUserDetails
	Creator     database.User
}

// NewClient returns a Client you can use to call methods on a Server listening
// at the given domain:port url.
//
// Provide a non-blank path to a certificate to force us to trust that
// certificate, eg. if the server was started with a self-signed certificate.
//
// Optionally provide a username and password in order to login.
// This will check the user has the correct authority in a JWT.
func NewClient(url, cert string, userpass ...string) (*Client, error) {
	c, err := gas.NewClientCLI(".tt.jwt", serverTokenBasename, url, cert, false)
	if err != nil {
		return nil, err
	}

	if errc := c.Login(userpass...); errc != nil {
		return nil, errc
	}

	jwt, err := c.GetJWT()
	if err != nil {
		return nil, err
	}

	return &Client{
		url:  url,
		cert: cert,
		jwt:  jwt,
	}, nil
}

func (c *Client) request() *resty.Request {
	return gas.NewAuthenticatedClientRequest(c.url, c.cert, c.jwt)
}

// CreateUser adds the given user to the database.
func (c *Client) CreateUser(u *database.User) error {
	resp, err := c.request().ForceContentType("application/json").SetBody(u).Post(EndPointAuthUser)
	if err != nil {
		return err
	}

	if resp.StatusCode() != http.StatusNoContent {
		return errors.New(resp.String()) //nolint:err113
	}

	return nil
}

// GetUser checks that a user exists in the database.
func (c *Client) GetUserByName(username string) (*database.User, error) {
	user := &database.User{}

	resp, err := c.request().SetHeader("Accept", "application/json").SetQueryParam("name", username).
		SetResult(user).
		Get(EndPointAuthUser)
	if err != nil {
		return nil, err
	}

	return user, responseToErr(resp)
}

// putObject sends obj encoded as JSON in the body via a PUT to the given url.
// If optionalResponseObj is defined, gets that decoded from the JSON response.
func (c *Client) putObject(url string, obj interface{}, optionalResponseObj ...interface{}) (*resty.Response, error) { //nolint:unused,lll
	req := c.setBodyAndOptionalResult(obj, optionalResponseObj...)

	resp, err := req.Put(url)
	if err != nil {
		return nil, err
	}

	return resp, responseToErr(resp)
}

func (c *Client) setBodyAndOptionalResult(thing interface{}, optionalResponseObj ...interface{}) *resty.Request { //nolint:unused,lll
	req := c.request().ForceContentType("application/json").SetBody(thing)

	if len(optionalResponseObj) == 1 {
		req = req.SetResult(&optionalResponseObj[0])
	}

	return req
}

// responseToErr converts a response's status code to one of our errors, or nil
// if there's no problem.
func responseToErr(resp *resty.Response) error {
	var err error

	switch resp.StatusCode() {
	case http.StatusUnauthorized:
		err = gas.ErrNoAuth
	case http.StatusBadRequest:
		err = ErrInvalidInput
	case http.StatusNotFound:
		err = gas.ErrNeedsAuth
	case http.StatusOK:
		return nil
	default:
		err = ErrInternal
	}

	body := string(resp.Body())

	if body != "" {
		err = fmt.Errorf("%w: %s", err, body)
	}

	return err
}

// PostThing posts a thing to the database.
func (c *Client) PostThing(t *database.CreateThingParams) error {
	resp, err := c.request().ForceContentType("application/json").SetBody(t).Post(EndPointAuthThings)
	if err != nil {
		return err
	}

	if resp.StatusCode() != http.StatusNoContent {
		return errors.New(resp.String()) //nolint:err113
	}

	return nil
}

// GetSubscribers gets the subscribers to a thing given the Thing ID.
func (c *Client) GetSubscribers(id uint32) ([]database.Subscriber, error) {
	subscribers := []database.Subscriber{}

	resp, err := c.request().SetHeader("Accept", "application/json").
		SetQueryParam("id", strconv.Itoa(int(id))).
		SetResult(&subscribers).
		Get(EndPointAuthSubscribers)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, errors.New(resp.String()) //nolint:err113
	}

	return subscribers, responseToErr(resp)
}

// GetUserByID gets a user given the user ID.
func (c *Client) GetUserByID(id uint32) (*database.User, error) {
	user := &database.User{}

	resp, err := c.request().SetHeader("Accept", "application/json").
		SetQueryParam("id", strconv.Itoa(int(id))).
		SetResult(&user).
		Get(EndPointAuthUser)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, errors.New(resp.String()) //nolint:err113
	}

	return user, responseToErr(resp)
}

// GetThings gets optionally filtered Things from the database.
func (c *Client) GetThings(filter *database.GetThingsParams) ([]database.Thing, error) {
	things := []database.Thing{}

	resp, err := c.request().SetHeader("Accept", "application/json").SetQueryParams(map[string]string{
		"type":     string(filter.FilterOnType),
		"page":     strconv.Itoa(filter.Page),
		"per_page": strconv.Itoa(filter.ThingsPerPage),
		"dir":      string(filter.OrderDirection),
		"sort":     string(filter.OrderBy),
	}).
		SetResult(&things).
		Get(EndPointAuthThings)
	if err != nil {
		return nil, err
	}

	return things, responseToErr(resp)
}

// GetThingByID returns a thing given an ID.
func (c *Client) GetThingByID(id uint32) (*database.Thing, error) {
	thing := &database.Thing{}

	resp, err := c.request().SetHeader("Accept", "application/json").SetQueryParam("id", strconv.Itoa(int(id))).
		SetResult(&thing).
		Get(EndPointAuthThingID)
	if err != nil {
		return nil, err
	}

	return thing, responseToErr(resp)
}

// SubscriberAndserDetails gets the subscribers to a thing given the Thing ID.
func (c *Client) GetSubscriberAndUserDetails(sub *database.Subscriber) (*SubscriberWithUserDetails, error) {
	detailedSub := &SubscriberWithUserDetails{
		Creator: sub.Creator,
	}

	user, err := c.GetUserByID(sub.UserID)
	if err != nil {
		return nil, err
	}

	detailedSub.User = *user

	return detailedSub, err
}

// GetThingAndSubscriberDetails returns a struct with a Thing and
// information about its subscribers given.
func (c *Client) GetThingAndSubscriberDetails(tID uint32) (*ThingWithSubscriberDetails, error) {
	thing, err := c.GetThingByID(tID)
	if err != nil {
		return nil, err
	}

	subs, err := c.GetSubscribers(tID)
	if err != nil {
		return nil, err
	}

	var creator database.User

	detailedSubs := []SubscriberWithUserDetails{}

	for _, sub := range subs {
		detailedSub, errc := c.GetSubscriberAndUserDetails(&sub)
		if errc != nil {
			return nil, errc
		}

		detailedSubs = append(detailedSubs, *detailedSub)
		if detailedSub.Creator {
			creator = detailedSub.User
		}
	}

	return &ThingWithSubscriberDetails{Thing: *thing, Subscribers: detailedSubs, Creator: creator}, err
}

// getObj gets obj decoded from JSON from the given url.
func (c *Client) getObj(url string, obj interface{}) error { //nolint:unused
	resp, err := c.request().ForceContentType("application/json").
		SetResult(&obj).
		Get(url)
	if err != nil {
		return err
	}

	return responseToErr(resp)
}
