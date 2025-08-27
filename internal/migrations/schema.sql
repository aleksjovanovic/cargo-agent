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

CREATE TABLE IF NOT EXISTS countries (
    id SERIAL PRIMARY KEY,
    name VARCHAR(150) NOT NULL,
    code CHAR(2) NOT NULL,          -- ISO alpha-2
    alpha3_code CHAR(3) NOT NULL,   -- ISO alpha-3
    eu_member BOOLEAN DEFAULT FALSE,
    continent VARCHAR(50) DEFAULT 'Europe'
);

CREATE TABLE IF NOT EXISTS cities (
    id SERIAL PRIMARY KEY,
    name VARCHAR(150) NOT NULL,
    country_id INT NOT NULL REFERENCES countries(id) ON DELETE CASCADE
);

INSERT INTO countries (name, code, alpha3_code, eu_member) VALUES
('Albania', 'AL', 'ALB', FALSE),
('Andorra', 'AD', 'AND', FALSE),
('Austria', 'AT', 'AUT', TRUE),
('Belarus', 'BY', 'BLR', FALSE),
('Belgium', 'BE', 'BEL', TRUE),
('Bosnia and Herzegovina', 'BA', 'BIH', FALSE),
('Bulgaria', 'BG', 'BGR', TRUE),
('Croatia', 'HR', 'HRV', TRUE),
('Cyprus', 'CY', 'CYP', TRUE),
('Czechia', 'CZ', 'CZE', TRUE),
('Denmark', 'DK', 'DNK', TRUE),
('Estonia', 'EE', 'EST', TRUE),
('Finland', 'FI', 'FIN', TRUE),
('France', 'FR', 'FRA', TRUE),
('Germany', 'DE', 'DEU', TRUE),
('Greece', 'GR', 'GRC', TRUE),
('Hungary', 'HU', 'HUN', TRUE),
('Iceland', 'IS', 'ISL', FALSE),
('Ireland', 'IE', 'IRL', TRUE),
('Italy', 'IT', 'ITA', TRUE),
('Latvia', 'LV', 'LVA', TRUE),
('Liechtenstein', 'LI', 'LIE', FALSE),
('Lithuania', 'LT', 'LTU', TRUE),
('Luxembourg', 'LU', 'LUX', TRUE),
('Malta', 'MT', 'MLT', TRUE),
('Moldova', 'MD', 'MDA', FALSE),
('Monaco', 'MC', 'MCO', FALSE),
('Montenegro', 'ME', 'MNE', FALSE),
('Netherlands', 'NL', 'NLD', TRUE),
('North Macedonia', 'MK', 'MKD', FALSE),
('Norway', 'NO', 'NOR', FALSE),
('Poland', 'PL', 'POL', TRUE),
('Portugal', 'PT', 'PRT', TRUE),
('Romania', 'RO', 'ROU', TRUE),
('San Marino', 'SM', 'SMR', FALSE),
('Serbia', 'RS', 'SRB', FALSE),
('Slovakia', 'SK', 'SVK', TRUE),
('Slovenia', 'SI', 'SVN', TRUE),
('Spain', 'ES', 'ESP', TRUE),
('Sweden', 'SE', 'SWE', TRUE),
('Switzerland', 'CH', 'CHE', FALSE),
('Turkey', 'TR', 'TUR', FALSE),
('Ukraine', 'UA', 'UKR', FALSE),
('United Kingdom', 'GB', 'GBR', FALSE),
('Vatican City', 'VA', 'VAT', FALSE);

INSERT INTO cities (name, country_id) VALUES
('Tirana', 1),
('Andorra la Vella', 2),
('Vienna', 3),
('Minsk', 4),
('Brussels', 5),
('Sarajevo', 6),
('Sofia', 7),
('Zagreb', 8),
('Nicosia', 9),
('Prague', 10),
('Copenhagen', 11),
('Tallinn', 12),
('Helsinki', 13),
('Paris', 14),
('Berlin', 15),
('Athens', 16),
('Budapest', 17),
('Reykjavik', 18),
('Dublin', 19),
('Rome', 20),
('Riga', 21),
('Vaduz', 22),
('Vilnius', 23),
('Luxembourg', 24),
('Valletta', 25),
('Chișinău', 26),
('Monaco', 27),
('Podgorica', 28),
('Amsterdam', 29),
('Skopje', 30),
('Oslo', 31),
('Warsaw', 32),
('Lisbon', 33),
('Bucharest', 34),
('San Marino', 35),
('Belgrade', 36),
('Bratislava', 37),
('Ljubljana', 38),
('Madrid', 39),
('Stockholm', 40),
('Bern', 41),
('Ankara', 42),
('Kyiv', 43),
('London', 44),
('Vatican City', 45);
