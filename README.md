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

To start the server you'll need a certificate and key file, and to specify the
bind address. You can also define these as environment variables TT_SERVER_URL,
TT_SERVER_CERT and TT_SERVER_KEY in an env file.

## Development

Put your MySQL connection detail export statements in a `.env.development.local`
file in the repo folder (which won't get added to the repo). To actually run the
the MySQL tests you must also include TT_SQL_DO_TESTS=TABLES_WILL_BE_DROPPED in
that file.

NB: the database configured there will have its tables dropped and recreated at
the start of running tests!

To initialise a database, for now you can manually run database/mysql/schema.sql
against your database. NB: it will first drop all tables in the database!

For convenience, install air for automatic re-builds and server restarting when
you make changes to files:

```
go install github.com/air-verse/air@latest
```

Then you can bring up a development server that logs to STDERR like this:

```
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -sha256 -days 365 -subj '/CN=yourhost' -addext "subjectAltName = DNS:yourhost" -nodes

export TT_ENV=development
air server --url :4563 --cert cert.pem --key key.pem --logfile /root/uncreatable-file-path
```
