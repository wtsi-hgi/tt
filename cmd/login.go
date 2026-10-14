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

package cmd

import (
	"fmt"
	"net"

	ldap "github.com/go-ldap/ldap/v3"
	gas "github.com/wtsi-hgi/go-authserver"
)

// checkPassword defers to checkLDAPPassword(). Warns if we don't have the ldap
// details we need, and uses an always true password checker.
func createCheckPassword(serverLDAPFQDN, serverLDAPBindDN string) gas.AuthCallback {
	if serverLDAPFQDN == "" || serverLDAPBindDN == "" {
		return fakePasswordCheck
	}

	return func(username, password string) (bool, string) {
		return checkLDAPPassword(username, password, serverLDAPFQDN, serverLDAPBindDN)
	}
}

// fakePasswordCheck is for when we don't have ldap credentials, and are just
// testing. It always returns true, unless the username doesn't exist at all.
func fakePasswordCheck(username, _ string) (bool, string) {
	uid, err := gas.UserNameToUID(username)
	if err != nil {
		return false, ""
	}

	return true, uid
}

// checkLDAPPassword checks with LDAP if the given password is valid for the
// given username. Returns true if valid, along with the user's UID.
func checkLDAPPassword(username, password, serverLDAPFQDN, serverLDAPBindDN string) (bool, string) {
	uid, err := gas.UserNameToUID(username)
	if err != nil {
		return false, ""
	}

	// l, err := ldap.DialURL("ldaps://" + net.JoinHostPort(serverLDAPFQDN, "636"))
	l, err := ldap.DialURL("ldaps://" + net.JoinHostPort(serverLDAPFQDN, "636"))
	if err != nil {
		return false, ""
	}

	err = l.Bind(fmt.Sprintf(serverLDAPBindDN, username), password)
	if err != nil {
		return false, ""
	}

	return true, uid
}
