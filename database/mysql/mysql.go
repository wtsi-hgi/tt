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

package mysql

import (
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"regexp"
	"time"

	gsdmysql "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

//go:embed schema.sql
var schemaSQL string

const (
	sqlDriverName   = "mysql"
	sqlNetwork      = "tcp"
	connMaxLifetime = time.Minute * 3
	maxOpenConns    = 10
	maxIdleConns    = 10

	envVarEnv    = "TT_ENV"
	envVarUser   = "TT_SQL_USER"
	envVarPass   = "TT_SQL_PASS"
	envVarHost   = "TT_SQL_HOST"
	envVarPort   = "TT_SQL_PORT"
	envVarDBName = "TT_SQL_DB"
)

type Error string

func (e Error) Error() string { return string(e) }

const ErrMissingEnvs = Error("missing required environment variables")

// ConfigFromEnv returns a new MySQL config suitable for passing to New(),
// populated from environment variables TT_SQL_USER, TT_SQL_PASS, TT_SQL_HOST,
// TT_SQL_PORT, and TT_SQL_DB.
//
// If these environment variables are defined in a file called
// .env.development.local (and not previously defined elsewhere), they will be
// loaded only if TT_ENV is set to "development".
//
// If these environment variables are defined in a file called .env.test.local
// (and not previously defined elsewhere), they will be loaded only if TT_ENV is
// set to "test".
//
// If these environment variables are defined in a file called
// .env.production.local (and not previously defined elsewhere), they will be
// loaded only if TT_ENV is set to "production".
//
// If these environment variables are defined in a .env file (and not previously
// defined elsewhere), they will be automatically loaded.
//
// Optionally supply a directory to look for the .env* files in.
func ConfigFromEnv(dir ...string) (*gsdmysql.Config, error) {
	var parentDir string
	if len(dir) == 1 {
		parentDir = dir[0] + string(os.PathSeparator)
	}

	env := os.Getenv(envVarEnv)
	godotenv.Load(parentDir + ".env." + env + ".local")
	godotenv.Load(parentDir + ".env")

	user := os.Getenv(envVarUser)
	pass := os.Getenv(envVarPass)
	host := os.Getenv(envVarHost)
	port := os.Getenv(envVarPort)
	dbname := os.Getenv(envVarDBName)

	if user == "" || pass == "" || host == "" || port == "" || dbname == "" {
		return nil, ErrMissingEnvs
	}

	conf := gsdmysql.NewConfig()
	conf.User = user
	conf.Passwd = pass
	conf.Net = sqlNetwork
	conf.Addr = fmt.Sprintf("%s:%s", host, port)
	conf.DBName = dbname
	conf.ParseTime = true

	return conf, nil
}

// MySQLDB implements the database interface by storing and retrieving info
// about things and users from a MySQL database.
type MySQLDB struct {
	pool *sql.DB
}

// New connects to the configured mysql server and returns a new MySQLDB that
// can perform queries for things and users.
func New(config *gsdmysql.Config) (*MySQLDB, error) {
	pool, err := sql.Open(sqlDriverName, config.FormatDSN())
	if err != nil {
		return nil, err
	}

	pool.SetConnMaxLifetime(connMaxLifetime)
	pool.SetMaxOpenConns(maxOpenConns)
	pool.SetMaxIdleConns(maxIdleConns)

	return &MySQLDB{pool: pool}, pool.Ping()
}

// Reset drops all tables and recreates them. Use with extreme caution!
func (m *MySQLDB) Reset() error {
	statements := regexp.MustCompile(`\n\s*\n`).Split(schemaSQL, -1)

	tx, err := m.pool.Begin()
	if err != nil {
		return err
	}

	for _, stmt := range statements {
		_, err = tx.Exec(stmt)
		if err != nil {
			tx.Rollback()

			return err
		}
	}

	return tx.Commit()
}

// Close closes the database connection. Not strictly necessary to call this.
func (m *MySQLDB) Close() error {
	return m.pool.Close()
}
