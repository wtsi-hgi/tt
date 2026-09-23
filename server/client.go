/*******************************************************************************
 * Copyright (c) 2026 Genome Research Ltd.
 *
 * Authors:
 *	- Sendu Bala <sb10@sanger.ac.uk>
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

	"github.com/go-resty/resty/v2"
	gas "github.com/wtsi-hgi/go-authserver"
	"github.com/wtsi-hgi/tt/database"
)

const (
	ErrInvalidInput = Error("Invalid input")
	ErrInternal     = Error("internal error")
)

// Client is used to interact with the Server over the network, with
// authentication.
type Client struct {
	url  string
	cert string
	jwt  string
	// logger log15.Logger
}

// NewClient returns a Client you can use to call methods on a Server listening
// at the given domain:port url.
//
// Provide a non-blank path to a certificate to force us to trust that
// certificate, eg. if the server was started with a self-signed certificate.
//
// You must first gas.GetJWT() to get a JWT that you must supply here.
// TODO: update the help text to explain we login and the optinoal user pass...
func NewClient(url, cert string, userpass ...string) (*Client, error) {
	c, err := gas.NewClientCLI(".tt.jwt", ".tt.token", url, cert, false)
	if err != nil {
		return nil, err
	}

	if errc := c.Login(userpass...); err != nil {
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
	//TODO: secure endpoint
	resp, err := c.request().ForceContentType("application/json").SetBody(u).Post("/user")
	if err != nil {
		return err
	}

	if resp.StatusCode() != http.StatusNoContent {
		return errors.New(resp.String()) //nolint:err113
	}

	return nil
}

// GetUser checks that a user exists in the database
func (c *Client) GetUserByName(username string) (*database.User, error) {
	user := &database.User{}

	resp, err := c.request().SetHeader("Accept", "application/json").SetQueryParam("name", username).
		SetResult(user).
		Get("/user")
	if err != nil {
		return nil, err
	}

	// resp, err := c.request().SetHeader("Accept", "application/json").SetQueryParam("name", username).
	// 	Get("/user")
	// if err != nil {
	// 	return nil, err
	// }

	// fmt.Println(string(resp.Body()))

	return user, responseToErr(resp)
}

// putObject sends obj encoded as JSON in the body via a PUT to the given url.
// If optionalResponseObj is defined, gets that decoded from the JSON response.
func (c *Client) putObject(url string, obj interface{}, optionalResponseObj ...interface{}) (error, *resty.Response) { //nolint:revive,unused
	req := c.setBodyAndOptionalResult(obj, optionalResponseObj...)

	resp, err := req.Put(url)
	if err != nil {
		return err, nil
	}

	return responseToErr(resp), resp
}

func (c *Client) setBodyAndOptionalResult(thing interface{}, optionalResponseObj ...interface{}) *resty.Request { //nolint:unused
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

// GetThings gets optionally filtered Things from the database.
func (c *Client) GetThings(filter interface{}) ([]*database.Thing, error) { //nolint:revive
	// ...
	// err := c.getObj(EndPointAuth..., &things)
	return nil, nil
}

// func (c *Client) GetThings(filter interface{}) ([]*database.Thing, error) {
// 	// ...
// 	// err := c.getObj(EndPointAuth..., &things)

// 	return nil, nil
// }

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
