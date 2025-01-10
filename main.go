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

// package main is the access point to our cmd package sub-commands.

package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-sql-driver/mysql"
	_ "github.com/joho/godotenv/autoload"

	"github.com/wtsi-hgi/tt/db"
)

const (
	sqlDriverName = "mysql"
	sqlNetwork    = "tcp"
)

func main() {
	config, err := getSQLConfigFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	if err := run(config); err != nil {
		log.Fatal(err)
	}
}

func getSQLConfigFromEnv() (*mysql.Config, error) {
	user := os.Getenv("TT_SQL_USER")
	pass := os.Getenv("TT_SQL_PASS")
	host := os.Getenv("TT_SQL_HOST")
	port := os.Getenv("TT_SQL_PORT")
	dbname := os.Getenv("TT_SQL_DB")

	if user == "" || pass == "" || host == "" || port == "" || dbname == "" {
		return nil, fmt.Errorf("missing required environment variables")
	}

	conf := mysql.NewConfig()
	conf.User = user
	conf.Passwd = pass
	conf.Net = sqlNetwork
	conf.Addr = fmt.Sprintf("%s:%s", host, port)
	conf.DBName = dbname
	conf.ParseTime = true

	return conf, nil
}

func run(config *mysql.Config) error {
	ctx := context.Background()

	fmt.Printf("dsn: %s\n", config.FormatDSN())

	sqlDB, err := sql.Open(sqlDriverName, config.FormatDSN())
	if err != nil {
		return err
	}

	queries := db.New(sqlDB)

	err = queries.Reset(ctx)
	if err != nil {
		return err
	}

	err = queries.ResetUsers(ctx)
	if err != nil {
		return err
	}

	result, err := queries.CreateUser(ctx, db.CreateUserParams{Name: "sb10", Email: "sb10@sanger.ac.uk"})
	if err != nil {
		return err
	}

	uid, err := result.LastInsertId()
	if err != nil {
		return err
	}

	result, err = queries.CreateThing(ctx, db.CreateThingParams{
		Address: "/a/file.txt",
		Type:    db.ThingsTypeFile,
		Created: time.Now(),
		Reason:  "hgi resource",
		Remove:  time.Now().Add(time.Hour * 25),
	})
	if err != nil {
		return err
	}

	tid, err := result.LastInsertId()
	if err != nil {
		return err
	}

	log.Println(result)

	_, err = queries.Subscribe(ctx, db.SubscribeParams{
		UserID:  uint32(uid),
		ThingID: uint32(tid),
	})
	if err != nil {
		return err
	}

	things, err := queries.ListThings(ctx)
	if err != nil {
		return err
	}

	log.Println(things)

	return nil
}
