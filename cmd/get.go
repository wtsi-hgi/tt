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

	"github.com/spf13/cobra"
	"github.com/wtsi-hgi/tt/database"
	"github.com/wtsi-hgi/tt/server"
)

// serverCmd represents the server command.
var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get Things",
	Long: `Get temporary things, optionally filtered and sorted.
get connects to the started server in order to retrieve data in a table. 
For example, 
tt get --url [] --cert [] 
Will retrieve all things in the things table. 

The --url is of the started tt server, including its port, and for it to work
with your --cert, you probably need to specify it as fqdn:port. 
--url defaults to the TT_SERVER_URL env var.
--cert defaults to the TT_SERVER_CERT env var.

You can add optional filtering:
The --type flag will retrieve all Things of the specified type.
The --orderBy flag will order the Things returned by the provided field.
The --orderDir flag will sort the Things returned by asc/desc.
The --page flag and --perPage flag can be used to reduce the list of things 
returned. These default to 1 and 50 respectively.

This command is usable by anyone. `,

	RunE: func(cmd *cobra.Command, args []string) error { //nolint: revive
		var url, cert, typ, orderBy, orderDir, page, perPage string

		for name, v := range map[string]*string{
			"url":      &url,  //nolint:goconst
			"cert":     &cert, //nolint:goconst
			"type":     &typ,
			"orderBy":  &orderBy,
			"orderDir": &orderDir,
			"page":     &page,
			"perPage":  &perPage,
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

		filter, err := database.NewGetThingsParams(typ, orderBy, orderDir, page, perPage)
		if err != nil {
			return err
		}

		things, err := client.GetThings(&filter)
		if err != nil {
			return err
		}

		if len(things) == 0 {
			warn("No Results")

			return nil
		}

		for _, thing := range things {
			thingWithSubs, err := client.GetThingAndSubscriberDetails(thing.ID)
			if err != nil {
				return err
			}

			cliPrint("%d\n%s\t%s\t%s\t%s\n", thing.ID, thing.Name, thing.Version, thing.Type, thingWithSubs.Creator.Name)
		}

		return nil
	},
}

func init() {
	RootCmd.AddCommand(getCmd)

	// flags specific to this sub-command
	getCmd.Flags().StringP("url", "u", os.Getenv(serverURLEnvKey),
		"tt server URL in the form host:port")
	getCmd.Flags().StringP("cert", "c", os.Getenv(serverCertEnvKey),
		"path to server certificate file")
	getCmd.MarkFlagRequired("url")  //nolint:errcheck
	getCmd.MarkFlagRequired("cert") //nolint:errcheck

	getCmd.Flags().StringP("type", "t", "",
		"type of things you would like to see")
	getCmd.Flags().StringP("orderBy", "o", "",
		"field you would like to order by")
	getCmd.Flags().StringP("orderDir", "d", "",
		"direction of order (asc/desc)")
	getCmd.Flags().StringP("page", "p", "1",
		"page you would like to see")
	getCmd.Flags().StringP("perPage", "i", "50",
		"number of things per page")
}
