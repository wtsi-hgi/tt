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
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	gas "github.com/wtsi-hgi/go-authserver"
	"github.com/wtsi-hgi/tt/database"
)

// serverCmd represents the server command.
var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Gewt a Thing",
	Long:  ``,

	RunE: func(cmd *cobra.Command, args []string) error { //nolint: revive
		var url, cert, typ string

		for name, v := range map[string]*string{
			"url":  &url,  //nolint:goconst
			"cert": &cert, //nolint:goconst
			"type": &typ,
		} {
			val, err := cmd.Flags().GetString(name)
			if err != nil {
				return err
			}

			*v = val
		}

		client := gas.NewClientRequest(url, cert)

		resp, err := client.SetQueryParam("type", typ).SetHeader("Accept", "application/json").Get("/things")
		if err != nil {
			return err
		}

		var things []database.Thing

		if err := json.Unmarshal(resp.Body(), &things); err != nil {
			return err
		}

		if len(things) == 0 {
			fmt.Println("No Results")

			return nil
		}

		for _, thing := range things {
			fmt.Printf("%s\t%s\n", thing.Name, thing.Version)
		}

		return nil
	},
}

func init() {
	RootCmd.AddCommand(getCmd)

	// flags specific to this sub-command
	getCmd.Flags().String("url", os.Getenv(serverURLEnvKey),
		"tt server URL in the form host:port")
	getCmd.Flags().String("cert", os.Getenv(serverCertEnvKey),
		"path to server certificate file")
	getCmd.MarkFlagRequired("url")  //nolint:errcheck
	getCmd.MarkFlagRequired("cert") //nolint:errcheck

	getCmd.Flags().String("type", "",
		"type of thing")
}
