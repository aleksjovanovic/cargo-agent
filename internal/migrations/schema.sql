DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_status') THEN
        CREATE TYPE user_status AS ENUM ('active', 'inactive', 'suspended', 'deleted', 'draft');
    END IF;
END
$$;

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(150) NOT NULL,
    email VARCHAR(254) NOT NULL, 
    password VARCHAR(255) NOT NULL,
    name VARCHAR(150) NOT NULL,
    country CHAR(2) NOT NULL,
    city VARCHAR(100) NOT NULL,
    legal_address VARCHAR(100) NOT NULL,
    vat_number VARCHAR(30) NOT NULL,
    status user_status NOT NULL DEFAULT 'draft',
    language CHAR(2) NOT NULL,
    created TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);