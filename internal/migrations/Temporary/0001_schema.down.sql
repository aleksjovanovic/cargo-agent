DROP INDEX IF EXISTS idx_cities_country_id;
DROP INDEX IF EXISTS idx_countries_alpha3;
DROP INDEX IF EXISTS idx_countries_code;

ALTER TABLE IF EXISTS cities DROP CONSTRAINT IF EXISTS cities_name_country_uk;
ALTER TABLE IF EXISTS countries DROP CONSTRAINT IF EXISTS countries_name_uk;
ALTER TABLE IF EXISTS countries DROP CONSTRAINT IF EXISTS countries_alpha3_uk;
ALTER TABLE IF EXISTS countries DROP CONSTRAINT IF EXISTS countries_code_uk;
ALTER TABLE IF EXISTS users DROP CONSTRAINT IF EXISTS users_email_uk;
ALTER TABLE IF EXISTS users DROP CONSTRAINT IF EXISTS users_username_uk;

DROP TABLE IF EXISTS cities;
DROP TABLE IF EXISTS countries;
DROP TABLE IF EXISTS users;

-- Enum može da se obriše tek kada više nema tabela koje ga koriste
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_status') THEN
        DROP TYPE user_status;
    END IF;
END
$$ LANGUAGE plpgsql;