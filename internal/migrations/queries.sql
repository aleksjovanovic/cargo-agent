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
     poster_name, price, currency, notes, status
) VALUES (
    sqlc.arg(created_by),
    sqlc.arg(origin_country_id), sqlc.arg(origin_city_id),
    sqlc.arg(destination_country_id), sqlc.arg(destination_city_id),
    sqlc.arg(loading_places), sqlc.arg(unloading_places),
    sqlc.arg(ready_to_load_by), sqlc.arg(delivery_deadline),
    sqlc.arg(load_type)::load_type,
    sqlc.arg(truck_type)::truck_type,
    sqlc.arg(weight_t), sqlc.arg(volume_m3), sqlc.arg(pallets), sqlc.arg(palletized),
    sqlc.arg(temperature_min_c), sqlc.arg(temperature_max_c),
    sqlc.arg(published_at), sqlc.arg(expires_at),
    sqlc.arg(poster_name), sqlc.arg(price), sqlc.arg(currency), sqlc.arg(notes),
    sqlc.arg(status)::offer_status
)
RETURNING
    id,
    created_by,
    origin_country_id, origin_city_id,
    destination_country_id, destination_city_id,
    loading_places, unloading_places,
    ready_to_load_by, delivery_deadline,
    load_type::text   AS load_type,
    truck_type::text  AS truck_type,
    weight_t, volume_m3, pallets, palletized,
    temperature_min_c, temperature_max_c,
    published_at, expires_at,
    price, currency, notes,
    status::text      AS status,
    created_at, updated_at;

-- name: GetCargoOffer :one
SELECT
    id,
    created_by,
    origin_country_id, origin_city_id,
    destination_country_id, destination_city_id,
    loading_places, unloading_places,
    ready_to_load_by, delivery_deadline,
    load_type::text   AS load_type,
    truck_type::text  AS truck_type,
    weight_t, volume_m3, pallets, palletized,
    temperature_min_c, temperature_max_c,
    published_at, expires_at,
    price, currency, notes,
    status::text      AS status,
    created_at, updated_at
FROM cargo_offers
WHERE id = $1
LIMIT 1;

-- name: ListCargoOffers :many
SELECT *
FROM cargo_offers
WHERE (sqlc.narg('origin_country_id')::int IS NULL OR origin_country_id = sqlc.narg('origin_country_id')::int)
  AND (sqlc.narg('origin_city_id')::int IS NULL OR origin_city_id = sqlc.narg('origin_city_id')::int)
  AND (sqlc.narg('destination_country_id')::int IS NULL OR destination_country_id = sqlc.narg('destination_country_id')::int)
  AND (sqlc.narg('destination_city_id')::int IS NULL OR destination_city_id = sqlc.narg('destination_city_id')::int)
  AND (sqlc.narg('load_type')::text IS NULL OR load_type = sqlc.narg('load_type')::load_type)
  AND (sqlc.narg('truck_type')::text IS NULL OR truck_type = sqlc.narg('truck_type')::truck_type)
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::offer_status)
  AND (sqlc.narg('ready_from')::timestamptz IS NULL OR ready_to_load_by   >= sqlc.narg('ready_from')::timestamptz)
  AND (sqlc.narg('ready_to')::timestamptz   IS NULL OR ready_to_load_by   <= sqlc.narg('ready_to')::timestamptz)
  AND (sqlc.narg('delivery_from')::timestamptz IS NULL OR delivery_deadline >= sqlc.narg('delivery_from')::timestamptz)
  AND (sqlc.narg('delivery_to')::timestamptz   IS NULL OR delivery_deadline <= sqlc.narg('delivery_to')::timestamptz)
ORDER BY created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: UpdateCargoOfferStatus :one
UPDATE cargo_offers
SET status = $2, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: CreateTruckAvailability :one
INSERT INTO truck_availability (
  created_by,
  start_country_id, start_city_id,
  end_country_id, end_city_id,
  available_from, available_to,
  truck_type, max_weight_t, max_volume_m3,
  full_load, partial_load,
  loading_places, unloading_places,
  published_at, expires_at,
  poster_name, price_per_km, currency, notes, status
) VALUES (
  $1,
  $2, $3,
  $4, $5,
  $6, $7,
  $8, $9, $10,
  $11, $12,
  $13, $14,
  $15, $16,
  $17, $18, $19, $20, $21
)
RETURNING *;

-- name: GetTruckAvailability :one
SELECT * FROM truck_availability WHERE id = $1 LIMIT 1;

-- name: ListTruckAvailability :many
SELECT *
FROM truck_availability
WHERE
  (sqlc.narg('start_country_id')::int IS NULL OR start_country_id = sqlc.narg('start_country_id')::int)
  AND (sqlc.narg('start_city_id')::int IS NULL OR start_city_id = sqlc.narg('start_city_id')::int)
  AND (sqlc.narg('end_country_id')::int IS NULL OR end_country_id = sqlc.narg('end_country_id')::int)
  AND (sqlc.narg('end_city_id')::int IS NULL OR end_city_id = sqlc.narg('end_city_id')::int)

  -- ⬇️ ključna izmena: radimo TEXT poređenje, ne enum kast
  AND (sqlc.narg('truck_type')::text   IS NULL OR truck_type::text   = sqlc.narg('truck_type')::text)
  AND (sqlc.narg('status')::text       IS NULL OR status::text       = sqlc.narg('status')::text)

  AND (sqlc.narg('available_from')::timestamptz IS NULL OR available_from >= sqlc.narg('available_from')::timestamptz)
  AND (sqlc.narg('available_to')::timestamptz   IS NULL OR available_to   <= sqlc.narg('available_to')::timestamptz)
  AND (sqlc.narg('full_load')::bool    IS NULL OR full_load    = sqlc.narg('full_load')::bool)
  AND (sqlc.narg('partial_load')::bool IS NULL OR partial_load = sqlc.narg('partial_load')::bool)
ORDER BY published_at DESC NULLS LAST, id DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: UpdateTruckAvailabilityStatus :one
UPDATE truck_availability
SET status = $2, updated_at = now()
WHERE id = $1
RETURNING *;


