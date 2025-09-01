-- name: CreateUser :one
INSERT INTO users(username, email, password, name, country, city, legal_address, vat_number, language, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING id, username, email, name, country, city, legal_address, vat_number, status, language, created_at, updated_at;

-- name: CreateEmailVerificationToken :one
INSERT INTO email_verification_tokens (user_id, token, valid_until)
VALUES ($1, $2, $3)
RETURNING id, user_id, token, created_at, valid_until;

-- name: GetEmailVerificationToken :one
SELECT id, user_id, token, created_at, valid_until
FROM email_verification_tokens
WHERE token = $1
LIMIT 1;

-- name: DeleteEmailVerificationToken :exec
DELETE FROM email_verification_tokens
WHERE token = $1;

-- name: UpdateUserStatus :exec
UPDATE users
SET status = $2, updated_at = now()
WHERE id = $1;

-- name: DeleteExpiredEmailVerificationTokens :exec
DELETE FROM email_verification_tokens
WHERE valid_until < now();

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

-- name: CreateCargoOffer :one
INSERT INTO cargo_offers (
    created_by,
    origin_country_id, origin_city_id,
    destination_country_id, destination_city_id,
    loading_places, unloading_places,
    ready_to_load_by, delivery_deadline,
    load_type, truck_type,
    weight_t, volume_m3, pallets, palletized,
    temperature_min_c, temperature_max_c,
    published_at, expires_at,
    price, currency, notes, status
) VALUES (
    $1,
    $2, $3,
    $4, $5,
    $6, $7,
    $8, $9,
    $10, $11,
    $12, $13, $14, $15,
    $16, $17,
    $18, $19,
    $20, $21, $22, $23
)
RETURNING *;

-- name: GetCargoOffer :one
SELECT * FROM cargo_offers WHERE id = $1 LIMIT 1;
