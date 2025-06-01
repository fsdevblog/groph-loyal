-- name: Users_Create :one
INSERT INTO users
    (username, password)
VALUES ($1, $2)
RETURNING *;