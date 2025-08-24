-- name: CreateUser :one
INSERT INTO users(username, email, password, name, country, city, legal_address, vat_number, status, language, created, updated)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING id, username, email, name, country, city, legal_address, vat_number, status, language, created, updated;

-- name: GetUser :one
SELECT id, username, email, name, country, city, legal_address, vat_number, status, language, created, updated
FROM users
WHERE id = $1;

-- name: GetUserByUsernameOrEmail :one
SELECT id, username, password, email, name, country, city, legal_address, vat_number, status, language, created, updated
FROM users
WHERE username = $1 OR email=$1;

-- name: ListUsers :many 
SELECT id, username, email, name, country, city, legal_address, vat_number, status, language, created, updated
FROM users
ORDER BY id;