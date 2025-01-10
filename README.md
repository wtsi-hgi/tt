# tt
Manage temporary things, such as files, dirs, files in irods and s3, and
openstack instances.

## Development

Install sqlc:

```
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

After changing schema.sql or query.sql, run this:

```
sqlc generate
```