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
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a Thing",
	Long: `Create a Thing.

A Thing is something of types that has been created for temporary usage.
It can be of the following types: 
'dir', 'file', 'irods', 'openstack', 's3', 'resource'

For example,
tt create 
			--url 			[]
			--cert			[]
			--name 			"item"
			--version"      "1.2"
			--description   "Something"
			--type          "s3"
			--requestSource "jira"
			--address       "s3://bucket/path"
			--reason       	"xyz"
			--creator     	"username"
This command will access the started server and, after verification,
result in "item" being added to the table of Things with 
various information from the various mandatory fields supplied by the flag. 

The --url of the started tt server, including its port, and for it to work
with your --cert, you probably need to specify it as
fqdn:port. --url defaults to the TT_SERVER_URL env var. --cert 
defaults to the TT_SERVER_CERT env var.

The values provided would indicated the temporary Thing "item", of version 
"1.2" is of the type "s3". It does "something" and was requested through "jira" 
because "xyz" and can be found in the directory "s3://bucket/path".
The type of thing you supply will be validated.

If applicable, the following flags can also be supplied with strings:
--downloadMethod, --downloadURL, --license 
	These fields are specific the resource type
--created, --delete, 
	These are fields that may not be known.
	If a value is not supplied for the creation date, it will default to now,
	If a value is not supplied for the deletion date, 
	it will default to five years from the creation date.
	
This command can only be used by the person who started the server.
The creator field will default to the user who started the server 
	`,

	RunE: func(cmd *cobra.Command, args []string) error { //nolint: revive
		var url, cert, typ, creationDate, remove string

		var thing database.CreateThingParams

		for name, v := range map[string]*string{
			"url":            &url,  //nolint:goconst
			"cert":           &cert, //nolint:goconst
			"name":           &thing.Name,
			"description":    &thing.Description,
			"version":        &thing.Version,
			"type":           &typ,
			"address":        &thing.Address,
			"reason":         &thing.Reason,
			"requestSource":  &thing.RequestSource,
			"downloadMethod": &thing.DownloadMethod,
			"downloadURL":    &thing.URL,
			"created":        &creationDate,
			"delete":         &remove,
			"license":        &thing.License,
			"creator":        &thing.Creator,
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

		thing.Type, err = database.NewThingsType(typ)
		if err != nil {
			return err
		}

		err = client.PostThing(&thing)
		if err != nil {
			return err
		}

		return nil
	},
}

func init() { //nolint:funlen
	RootCmd.AddCommand(createCmd)

	// flags specific to this sub-command
	createCmd.Flags().String("url", os.Getenv(serverURLEnvKey),
		"tt server URL in the form host:port")
	createCmd.Flags().String("cert", os.Getenv(serverCertEnvKey),
		"path to server certificate file")
	createCmd.MarkFlagRequired("url")  //nolint:errcheck
	createCmd.MarkFlagRequired("cert") //nolint:errcheck

	createCmd.Flags().String("address", "",
		"address of thing")
	createCmd.Flags().String("type", "",
		"type of thing")
	createCmd.Flags().String("description", "",
		"description of thing")
	createCmd.Flags().String("name", "",
		"name of thing")
	createCmd.Flags().String("reason", "",
		"reason for thing existing")
	createCmd.Flags().String("requestSource", "",
		"source of thing")
	createCmd.Flags().String("version", "",
		"version of thing")
	createCmd.Flags().String("downloadMethod", "",
		"download method of thing")
	createCmd.Flags().String("downloadURL", "",
		"url of thing")
	createCmd.Flags().String("created", "",
		"creation date of thing")
	createCmd.Flags().String("delete", "",
		"deletion date of thing")
	createCmd.Flags().String("license", "",
		"license of thing")
	createCmd.Flags().String("creator", os.Getenv("USER"),
		"creator of thing")
	createCmd.MarkFlagRequired("address")       //nolint:errcheck
	createCmd.MarkFlagRequired("type")          //nolint:errcheck
	createCmd.MarkFlagRequired("description")   //nolint:errcheck
	createCmd.MarkFlagRequired("name")          //nolint:errcheck
	createCmd.MarkFlagRequired("reason")        //nolint:errcheck
	createCmd.MarkFlagRequired("requestSource") //nolint:errcheck
	createCmd.MarkFlagRequired("version")       //nolint:errcheck
}
