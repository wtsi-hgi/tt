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

package cmd

import (
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/wtsi-hgi/tt/database"
	"github.com/wtsi-hgi/tt/server"
)

// serverCmd represents the server command.
var unsubscribe = &cobra.Command{
	Use:   "unsubscribe",
	Short: "Unsubscribe to a Thing",
	Long: `Unsubscribe to a Thing.
The tt unsubscribe command is used to remove a subscription to the subscriptions table.
For example,
tt unsubscribe --url [] --cert [] --user "username" --id 1
Will unsubscribe the user "username" to the Thing that has the id "1". Leaving the 
--user flag blank will default it to the current CLI user. 

The --url of the started tt server, including its port, and for it to work
with your --cert, you probably need to specify it as fqdn:port. 
--url defaults to the TT_SERVER_URL env var. 
--cert defaults to the TT_SERVER_CERT env var.

This command is usable by anyone. However, if you are not the person
who started the server, you are not be able to unsubscribe other users to Things.`,

	RunE: func(cmd *cobra.Command, args []string) error { //nolint: revive
		var url, cert, id string

		var subPost database.NewSubscriptionParams
		for name, v := range map[string]*string{
			"url":  &url,              //nolint:goconst
			"cert": &cert,             //nolint:goconst
			"user": &subPost.Username, //nolint:goconst
			"id":   &id,
		} {
			val, err := cmd.Flags().GetString(name)
			if err != nil {
				return err
			}

			*v = val
		}

		client, err := server.NewClient(url, cert)
		if err != nil {
			return err
		}

		thingID, err := strconv.Atoi(id)
		if err != nil {
			return err
		}

		subPost.ThingID = uint32(thingID)

		return client.Unsubscribe(&subPost)
	},
}

func init() {
	RootCmd.AddCommand(unsubscribe)

	// flags specific to this sub-command
	unsubscribe.Flags().StringP("url", "u", os.Getenv(serverURLEnvKey),
		"tt server URL in the form host:port")
	unsubscribe.Flags().StringP("cert", "c", os.Getenv(serverCertEnvKey),
		"path to server certificate file")
	unsubscribe.MarkFlagRequired("url")  //nolint:errcheck
	unsubscribe.MarkFlagRequired("cert") //nolint:errcheck

	unsubscribe.Flags().StringP("user", "n", os.Getenv("USER"),
		"username of user")
	unsubscribe.Flags().StringP("id", "i", "",
		"id of thing")

	unsubscribe.MarkFlagRequired("id") //nolint:errcheck
}
