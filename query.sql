-- name: GetThingByID :one
SELECT * FROM things
WHERE id = ? LIMIT 1;

-- name: GetThingByAddressType :one
SELECT * FROM things
WHERE address = ? AND type = ? LIMIT 1;

-- name: ListThings :many
SELECT * FROM things
ORDER BY remove DESC;

-- name: ListThingsByType :many
SELECT * FROM things
WHERE type = ?
ORDER BY address;

-- name: CreateThing :execresult
INSERT INTO things (
  address, type, created, description, reason, remove
) VALUES (
  ?, ?, ?, ?, ?, ?
);

-- name: UpdateDescription :execresult
UPDATE things
SET description = ?
WHERE id = ?;

-- name: ExtendRemoval :execresult
UPDATE things
SET remove = ?, warned1 = NULL, warned2 = NULL
WHERE id = ?;

-- name: FirstWarningSent :execresult
UPDATE things
SET warned1 = ?
WHERE id = ?;

-- name: SecondWarningSent :execresult
UPDATE things
SET warned2 = ?
WHERE id = ?;

-- name: DeleteThing :exec
DELETE FROM things
WHERE id = ?;

-- name: CreateUser :execresult
INSERT INTO users (
  name, email
) VALUES (
  ?, ?
);

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = ?;

-- name: Subscribe :execresult
INSERT INTO subscribers (
  user_id, thing_id
) VALUES (
  ?, ?
);

-- name: Unsubscribe :exec
DELETE FROM subscribers
WHERE user_id = ? AND thing_id = ?;

-- name: ListSubscriptions :many
SELECT * FROM subscribers
WHERE user_id = ?;

-- name: ListSubscribers :many
SELECT * FROM subscribers
WHERE thing_id = ?;
