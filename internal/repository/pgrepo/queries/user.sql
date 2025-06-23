-- name: Users_Create :one
INSERT INTO users
(username, encrypted_password)
VALUES ($1, $2)
RETURNING *;

-- name: Users_FindByUsername :one
SELECT * FROM users WHERE username = $1;