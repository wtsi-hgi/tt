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

## Development

Install sqlc:

```
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

After changing schema.sql or query.sql, run this:

```
sqlc generate
```

Put your MySQL connection detail export statements in a `.env` file in the repo
folder (which won't get added to the repo).
