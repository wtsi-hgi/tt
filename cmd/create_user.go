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

package cmd

import (
	"errors"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	gas "github.com/wtsi-hgi/go-authserver"
)

// serverCmd represents the server command.
var createUser = &cobra.Command{
	Use:   "createUser", //TODO: better name for this; user has to type this on the terminal, eg just "user"
	Short: "Create a User",
	Long: `
The tt createUser command is used to add a new user to the users table.
For example,
tt createUser --url [] --cert [] --user "username" --email "username@something.com"
Will retrieve all things in the things table.

The --url of the started tt server, including its port, and for it to work
with your --cert, you probably need to specify it as
fqdn:port. --url defaults to the TT_SERVER_URL env var. --cert 
defaults to the TT_SERVER_CERT env var.

This command is only usable by the user who created the server. Access has been limited as 
users are automatically created when they access our web server so user creation will only be used 
with caution, when neccessary, by admins. 
`,

	RunE: func(cmd *cobra.Command, args []string) error { //nolint: revive
		var url, cert, user, email string

		for name, v := range map[string]*string{
			"url":   &url,  //nolint:goconst
			"cert":  &cert, //nolint:goconst
			"user":  &user,
			"email": &email,
		} {
			val, err := cmd.Flags().GetString(name)
			if err != nil {
				return err
			}

			*v = val
		}

		client := gas.NewClientRequest(url, cert)

		resp, err := client.SetQueryParam("user", user).SetQueryParam("email", email).Post("/user")
		if err != nil {
			return err
		}

		if resp.StatusCode() != http.StatusOK {
			return errors.New(resp.String()) //nolint:err113
		}

		return nil
	},
}

func init() {
	RootCmd.AddCommand(createUser)

	// flags specific to this sub-command
	createUser.Flags().String("url", os.Getenv(serverURLEnvKey),
		"tt server URL in the form host:port")
	createUser.Flags().String("cert", os.Getenv(serverCertEnvKey),
		"path to server certificate file")
	createUser.MarkFlagRequired("url")  //nolint:errcheck
	createUser.MarkFlagRequired("cert") //nolint:errcheck

	createUser.Flags().String("user", "",
		"username of user")
	createUser.Flags().String("email", "",
		"email of user")
}
