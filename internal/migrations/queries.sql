-- name: CreateUser :one
INSERT INTO users(username, email, password, name, country, city, legal_address, vat_number, status, language, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING id, username, email, name, country, city, legal_address, vat_number, status, language, created_at, updated_at;

-- name: UpdateUserProfile :one
UPDATE users
SET
  username      = COALESCE(sqlc.narg('username'), username),
  email         = COALESCE(sqlc.narg('email'), email),
  name          = COALESCE(sqlc.narg('name'), name),
  country       = COALESCE(sqlc.narg('country'), country),
  city          = COALESCE(sqlc.narg('city'), city),
  legal_address = COALESCE(sqlc.narg('legal_address'), legal_address),
  vat_number    = COALESCE(sqlc.narg('vat_number'), vat_number),
  language      = COALESCE(sqlc.narg('language'), language),
  updated_at       = COALESCE(sqlc.narg('updated_at'), updated_at)
WHERE id = sqlc.arg('id')
AND status ='active'
RETURNING id, username, email, name, country, city, legal_address, vat_number, status, language, created_at, updated_at;

-- name: DeleteUser :exec
UPDATE users
SET status = 'deleted',
    deleted_at = $2
WHERE id = $1 AND status <> 'deleted';

-- name: GetUser :one
SELECT id, username, email, name, country, city, legal_address, vat_number, status, language, created_at, updated_at
FROM users
WHERE id = $1
AND status = 'active';

-- name: GetUserPassword :one
SELECT password
FROM users
WHERE id = $1
AND status = 'active';

-- name: ChangePassword :exec
UPDATE users
SET password=$2, updated_at=$3
WHERE id = $1;

-- name: GetUserByUsernameOrEmail :one
SELECT id, username, password, email, name, country, city, legal_address, vat_number, status, language, created_at, updated_at
FROM users
WHERE (username = $1 OR email=$1)
AND status = 'active';

-- name: ListUsers :many 
SELECT id, username, email, name, country, city, legal_address, vat_number, status, language, created_at, updated_at
FROM users
WHERE status = 'active'
ORDER BY id;

-- name: ListCountries :many 
SELECT id, name, code, alpha3_code, eu_member, continent
FROM countries
ORDER BY id;

-- name: GetCountryByID :one 
SELECT id, name, code, alpha3_code, eu_member, continent
FROM countries
WHERE id = $1;

-- name: GetCountryByName :one 
SELECT id, name, code, alpha3_code, eu_member, continent
FROM countries
WHERE Lower(name) = Lower($1);

-- name: ListCitiesByCountryID :many 
SELECT id, name, country_id, latitude, longitude
FROM cities
WHERE country_id=$1
ORDER BY name;