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
SELECT id, name, country_id
FROM cities
WHERE country_id=$1
ORDER BY name;