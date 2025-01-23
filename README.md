# tt
Manage temporary things, such as files, dirs, files in irods and s3, and
openstack instances.

## Deployment

You'll need a MySQL database and environment variables exported to configure
how to connect to your database:

```
export TT_SQL_HOST=localhost
export TT_SQL_PORT=3306
export TT_SQL_USER=user
export TT_SQL_PASS=pass
export TT_SQL_DB=tt_db
```

If you put these statements in a `.env` file that's in the current working
directory when you start the tt server, it will automatically be sourced.

To distinguish test and production environments, instead put the statements in
files `.env.test.local` and `.env.production.local` respectively, and then set
the TT_ENV environment variable to "test" or "production" to select the
environment to use. (`.env` files will still be loaded when TT_ENV is set, but
at a lower precedence than the local files.)

## Development

Install sqlc:

```
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

After changing schema.sql or query.sql, run this:

```
sqlc generate
```

Put your MySQL connection detail export statements in a `.env.development.local`
file in the repo folder (which won't get added to the repo). To actually run the
the MySQL tests you must also include TT_SQL_DO_TESTS=TABLES_WILL_BE_DROPPED in
that file.

NB: the database configured there will have its tables dropped and recreated at
the start of running tests!

Bring up a development server like this:

```
export TT_ENV=development
go run main.go serve --address :4563
```
