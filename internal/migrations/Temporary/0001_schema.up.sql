-- Enum za status korisnika (idempotentno)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_status') THEN
        CREATE TYPE user_status AS ENUM ('active', 'inactive', 'suspended', 'deleted', 'draft');
    END IF;
END
$$ LANGUAGE plpgsql;

-- users
CREATE TABLE IF NOT EXISTS users (
    id            SERIAL PRIMARY KEY,
    username      VARCHAR(150) NOT NULL,
    email         VARCHAR(254) NOT NULL,
    password      VARCHAR(255) NOT NULL,
    name          VARCHAR(150) NOT NULL,
    country       CHAR(2)      NOT NULL,
    city          VARCHAR(100) NOT NULL,
    legal_address VARCHAR(100) NOT NULL,
    vat_number    VARCHAR(30)  NOT NULL,
    status        user_status  NOT NULL DEFAULT 'draft',
    language      CHAR(2)      NOT NULL,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at    TIMESTAMPTZ  DEFAULT NULL
);

-- Jedinstvena ograničenja (odvojeno da lakše debuguješ konflikte)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'users_username_uk'
    ) THEN
        ALTER TABLE users ADD CONSTRAINT users_username_uk UNIQUE (username);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'users_email_uk'
    ) THEN
        ALTER TABLE users ADD CONSTRAINT users_email_uk UNIQUE (email);
    END IF;
END
$$ LANGUAGE plpgsql;

-- countries
CREATE TABLE IF NOT EXISTS countries (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(150) NOT NULL,
    code        CHAR(2)      NOT NULL,
    alpha3_code CHAR(3)      NOT NULL,
    eu_member   BOOLEAN               DEFAULT FALSE,
    continent   VARCHAR(50)           DEFAULT 'Europe'
);

-- Jedinstveni ključevi i indeksi
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'countries_code_uk') THEN
        ALTER TABLE countries ADD CONSTRAINT countries_code_uk UNIQUE (code);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'countries_alpha3_uk') THEN
        ALTER TABLE countries ADD CONSTRAINT countries_alpha3_uk UNIQUE (alpha3_code);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'countries_name_uk') THEN
        ALTER TABLE countries ADD CONSTRAINT countries_name_uk UNIQUE (name);
    END IF;
END
$$ LANGUAGE plpgsql;

CREATE INDEX IF NOT EXISTS idx_countries_code   ON countries(code);
CREATE INDEX IF NOT EXISTS idx_countries_alpha3 ON countries(alpha3_code);

-- cities
CREATE TABLE IF NOT EXISTS cities (
    id         SERIAL PRIMARY KEY,
    name       VARCHAR(150) NOT NULL,
    country_id INT          NOT NULL REFERENCES countries(id) ON DELETE CASCADE
);

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'cities_name_country_uk') THEN
        ALTER TABLE cities ADD CONSTRAINT cities_name_country_uk UNIQUE (name, country_id);
    END IF;
END
$$ LANGUAGE plpgsql;

CREATE INDEX IF NOT EXISTS idx_cities_country_id ON cities(country_id);