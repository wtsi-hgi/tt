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
	"io"
	"log/syslog"
	"time"

	"github.com/inconshreveable/log15"
	"github.com/spf13/cobra"
	"github.com/wtsi-hgi/tt/database/mysql"
	"github.com/wtsi-hgi/tt/server"
)

// options for this cmd.
var serverLogPath string
var serverKey string
var serverLDAPFQDN string
var serverLDAPBindDN string

// serverCmd represents the server command.
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the web server",
	Long: `Start the web server.

The tt web server is used to present a web interface to a MySQL database that
can record information about temporary things.

Your --url (in this context, think of it as the bind address) should include the
port; you probably need to specify it as fqdn:port. --url defaults to the
TT_SERVER_URL env var.

You will also need your database connection details in env vars:
export TT_SQL_HOST=localhost
export TT_SQL_PORT=3306
export TT_SQL_USER=user
export TT_SQL_PASS=pass
export TT_SQL_DB=tt_db

To automatically load the environment variables you can put them all in to a
.env file, or if you'll have multiple deployments, into files '.env.test.local'
and '.env.production.local', and then set the TT_ENV environment variable to
"test" or "production" to select the environment to use. ('.env' files will
still be loaded when TT_ENV is set, but at a lower precedence than the local
files.)

The server will log all messages (of any severity) to syslog at the INFO level,
except for non-graceful stops of the server, which are sent at the CRIT level or
include 'panic' in the message. The messages are tagged 'tt-server', and you
might want to filter away 'STATUS=200' to find problems.
If --logfile is supplied, logs to that file instead of syslog.

This command will block forever in the foreground; you can background it with
ctrl-z; bg. Or better yet, use the daemonize program to daemonize this.
`,
	Run: func(cmd *cobra.Command, args []string) {
		logWriter := setServerLogger(serverLogPath)

		config, err := mysql.ConfigFromEnv()
		if err != nil {
			die("failed to get database config: %s", err)
		}

		database, err := mysql.New(config)
		if err != nil {
			die("error opening database: %s", err)
		}

		conf := server.Config{
			HTTPLogger: logWriter,
			Database:   database,
		}

		s, err := server.New(conf)
		if err != nil {
			die("failed to configure server: %s", err)
		}

		defer s.Stop()

		sayStarted()

		err = s.Start(serverURL)
		if err != nil {
			die("non-graceful stop: %s", err)
		}
	},
}

func init() {
	RootCmd.AddCommand(serverCmd)

	// flags specific to this sub-command
	serverCmd.Flags().StringVar(&serverLogPath, "logfile", "",
		"log to this file instead of syslog")
}

// setServerLogger makes our appLogger log to the given path if non-blank,
// otherwise to syslog. Returns an io.Writer version of our appLogger for the
// server to log to.
func setServerLogger(path string) io.Writer {
	if path == "" {
		logToSyslog()
	} else {
		logToFile(path)
	}

	lw := &log15Writer{logger: appLogger}

	return lw
}

// logToSyslog sets our applogger to log to syslog, dies if it can't.
func logToSyslog() {
	fh, err := log15.SyslogHandler(syslog.LOG_INFO|syslog.LOG_DAEMON, "tt-server", log15.LogfmtFormat())
	if err != nil {
		die("failed to log to syslog: %s", err)
	}

	appLogger.SetHandler(fh)
}

// log15Writer wraps a log15.Logger to make it conform to io.Writer interface.
type log15Writer struct {
	logger log15.Logger
}

// Write conforms to the io.Writer interface.
func (w *log15Writer) Write(p []byte) (n int, err error) {
	w.logger.Info(string(p))

	return len(p), nil
}

// sayStarted logs to console that the server stated. It does this a second
// after being calling in a goroutine, when we can assume the server has
// actually started; if it failed, we expect it to do so in less than a second
// and exit.
func sayStarted() {
	<-time.After(1 * time.Second)

	info("server started")
}
