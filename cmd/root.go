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

// package cmd is the cobra file that enables subcommands and handles
// command-line args.

package cmd

import (
	"fmt"
	"os"

	"github.com/inconshreveable/log15"
	"github.com/spf13/cobra"
	"github.com/wtsi-hgi/tt/database/mysql"
)

// appLogger is used for logging events in our commands.
var appLogger = log15.New()

const (
	serverURLEnvKey  = "TT_SERVER_URL"
	serverCertEnvKey = "TT_SERVER_CERT"
	serverKeyEnvKey  = "TT_SERVER_KEY"
)

// global options.
var serverURL string
var serverKey string
var serverCert string

// RootCmd represents the base command when called without any subcommands.
var RootCmd = &cobra.Command{
	Use:   "tt",
	Short: "tt helps you keep track of and clean up temporary things",
	Long: `tt helps you keep track of and clean up temporary things.

TODO: help text
`,
}

// Execute adds all child commands to the root command and sets flags
// appropriately. This is called by main.main(). It only needs to happen once to
// the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		die("%s", err.Error())
	}
}

func init() {
	// set up logging to stderr
	appLogger.SetHandler(log15.LvlFilterHandler(log15.LvlInfo, log15.StderrHandler))

	mysql.ConfigFromEnv()

	// global flags
	RootCmd.PersistentFlags().StringVar(&serverURL, "url", os.Getenv(serverURLEnvKey),
		"tt server URL in the form host:port")
	RootCmd.PersistentFlags().StringVar(&serverCert, "cert", os.Getenv(serverCertEnvKey),
		"path to server certificate file")
	RootCmd.PersistentFlags().StringVar(&serverKey, "key", os.Getenv(serverKeyEnvKey),
		"path to server key file")
}

// ensureServerArgs dies if --url or --cert or --key have not been set.
func ensureServerArgs() {
	if serverURL == "" {
		die("you must supply --url")
	}

	if serverCert == "" {
		die("you must supply --cert")
	}

	if serverKey == "" {
		die("you must supply --key")
	}
}

// logToFile logs to the given file.
func logToFile(path string) {
	fh, err := log15.FileHandler(path, log15.LogfmtFormat())
	if err != nil {
		fh = log15.StderrHandler

		warn("can't write to log file; logging to stderr instead (%s)", err)
	}

	appLogger.SetHandler(fh)
}

// cliPrint outputs the message to STDOUT.
func cliPrint(msg string, a ...interface{}) {
	fmt.Fprintf(os.Stdout, msg, a...)
}

// cliPrintRaw is like cliPrint, but does no interpretation of placeholders in
// msg.
func cliPrintRaw(msg string) {
	fmt.Fprint(os.Stdout, msg)
}

// info is a convenience to log a message at the Info level.
func info(msg string, a ...interface{}) {
	appLogger.Info(fmt.Sprintf(msg, a...))
}

// warn is a convenience to log a message at the Warn level.
func warn(msg string, a ...interface{}) {
	appLogger.Warn(fmt.Sprintf(msg, a...))
}

// die is a convenience to log a message at the Error level and exit non zero.
func die(msg string, a ...interface{}) {
	appLogger.Error(fmt.Sprintf(msg, a...))
	os.Exit(1)
}
