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
	ErrInvalidInput     = Error("Invalid input")
	ErrInternal         = Error("internal error")
	serverTokenBasename = ".tt.token"
)

// Client is used to interact with the Server over the network, with
// authentication.
type Client struct {
	url  string
	cert string
	jwt  string
}

type ThingWithSubscriberDetails struct {
	database.Thing
	Subscribers []database.User
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
func NewClient(url, cert string) (*Client, error) {
	c, err := gas.NewClientCLI(".tt.jwt", serverTokenBasename, url, cert, false)
	if err != nil {
		return nil, err
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
func (c *Client) GetSubscribers(thingID uint32) ([]database.Subscriber, error) {
	subscribers := []database.Subscriber{}

	resp, err := c.request().SetHeader("Accept", "application/json").
		SetResult(&subscribers).
		Get(fmt.Sprintf("%s/%d", EndPointAuthSubscriber, thingID))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, errors.New(resp.String()) //nolint:err113
	}

	return subscribers, responseToErr(resp)
}

// TODO: post a subscriber.
func (c *Client) Subscribe(subscription *database.NewSubscriptionParams) error {
	resp, err := c.request().ForceContentType("application/json").SetBody(subscription).Post(EndPointAuthSubscriber)
	if err != nil {
		return err
	}

	if resp.StatusCode() != http.StatusNoContent {
		return errors.New(resp.String()) //nolint:err113
	}

	return nil
}

// TODO: delete a subscriber.
func (c *Client) Unsubscribe(subscription *database.NewSubscriptionParams) error { //pass in subscriber type?
	resp, err := c.request().ForceContentType("application/json").SetBody(subscription).Delete(EndPointAuthSubscriber)
	if err != nil {
		return err
	}

	if resp.StatusCode() != http.StatusNoContent {
		return errors.New(resp.String()) //nolint:err113
	}

	return nil
}

// GetSubscriptions returns a list of the Things that are subscribed to by a user given the username.
func (c *Client) GetSubscriptions(username string) ([]database.Thing, error) {
	things := []database.Thing{}

	resp, err := c.request().SetHeader("Accept", "application/json").SetQueryParam("username", username).
		SetResult(&things).
		Get(EndPointAuthSubscriber)
	if err != nil {
		return nil, err
	}

	return things, responseToErr(resp)
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

	resp, err := c.request().SetHeader("Accept", "application/json").
		SetResult(&thing).
		Get(fmt.Sprintf("%s/%d", EndPointAuthThings, id))
	if err != nil {
		return nil, err
	}

	return thing, responseToErr(resp)
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

	userSubs := []database.User{}

	for _, sub := range subs {
		user, errc := c.GetUserByID(sub.UserID)
		if errc != nil {
			return nil, errc
		}

		userSubs = append(userSubs, *user)
	}

	creator, errc := c.GetUserByID(thing.CreatorID)
	if errc != nil {
		return nil, errc
	}

	return &ThingWithSubscriberDetails{Thing: *thing, Subscribers: userSubs, Creator: *creator}, err
}
