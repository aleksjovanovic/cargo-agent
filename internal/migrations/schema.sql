DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_status') THEN
        CREATE TYPE user_status AS ENUM ('active', 'inactive', 'suspended', 'deleted', 'draft');
    END IF;
END
$$ LANGUAGE plpgsql;

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
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ DEFAULT NULL
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
    country_id INT          NOT NULL REFERENCES countries(id) ON DELETE CASCADE,
    latitude   DECIMAL(9, 6),
    longitude  DECIMAL(9, 6)
);

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'cities_name_country_uk') THEN
        ALTER TABLE cities ADD CONSTRAINT cities_name_country_uk UNIQUE (name, country_id);
    END IF;
END
$$ LANGUAGE plpgsql;

CREATE INDEX IF NOT EXISTS idx_cities_country_id ON cities(country_id);

-- Countries (idempotentno)
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
('Vatican City', 'VA', 'VAT', FALSE)
ON CONFLICT (code) DO NOTHING;

-- Cities (idempotentno)
INSERT INTO cities (name, country_id) VALUES
("Ada", 45.8025, 20.12583),
("Adorjan", 46.00333, 20.04007),
("Aleksandrovac", 43.45861, 21.0525),
("Aleksandrovo", 45.63755, 20.59288),
("Aleksinac", 43.54167, 21.70778),
("Alibunar", 45.08083, 20.96583),
("Apatin", 45.6726, 18.978),
("Aradac", 45.38346, 20.30137),
("Arangelovac", 44.30694, 20.56),
("Arilje", 43.75306, 20.09556),
("Bac", 45.39194, 19.23667),
("Bace", 43.21963, 21.34463),
("Backa Palanka", 45.24966, 19.39664),
("Backa Topola", 45.81516, 19.6318),
("Backi Breg", 45.92034, 18.92944),
("Backi Petrovac", 45.36056, 19.59167),
("Backo Gradiste", 45.53271, 20.03082),
("Backo Petrovo Selo", 45.70681, 20.07928),
("Badovinci", 44.78534, 19.37146),
("Bajina Basta", 43.97083, 19.5675),
("Banatska Dubica", 45.27146, 20.82833),
("Banatska Topola", 45.67248, 20.4653),
("Banatski Despotovac", 45.36606, 20.66407),
("Banatski Dvor", 45.51866, 20.51146),
("Banatski Karlovac", 45.04987, 21.018),
("Banatsko Karagorgevo", 45.58693, 20.56421),
("Banatsko Veliko Selo", 45.81961, 20.60772),
("Banovo Polje", 44.9104, 19.44916),
("Barajevo", 44.57889, 20.41583),
("Baranda", 45.08459, 20.44264),
("Baric", 44.6507, 20.25941),
("Barice", 45.18189, 21.08279),
("Basaid", 45.64102, 20.41434),
("Batocina", 44.15361, 21.08167),
("Bavaniste", 44.81907, 20.87654),
("Becej", 45.61632, 20.03331),
("Becmen", 44.77983, 20.20577),
("Bela Crkva", 44.8975, 21.41722),
("Bela Palanka", 43.21833, 22.31111),
("Belegis", 45.0192, 20.33323),
("Belgrade", 44.80401, 20.46513),
("Belo Blato", 45.27278, 20.375),
("Beloljin", 43.23846, 21.39673),
("Belotic", 44.81782, 19.54801),
("Belotic", 44.58099, 19.71932),
("Beocin", 45.20829, 19.72063),
("Besenovo", 45.08364, 19.70408),
("Beska", 45.13092, 20.06698),
("Biljaca", 42.3543, 21.7437),
("Blace", 43.29528, 21.28583),
("Bocar", 45.76994, 20.2839),
("Bogatic", 44.8375, 19.48056),
("Bogojevo", 45.53015, 19.13022),
("Bogosavac", 44.71799, 19.59533),
("Bojnik", 43.01224, 21.721),
("Boka", 45.3554, 20.82987),
("Boljevac", 43.83028, 21.95306),
("Boljevci", 44.72355, 20.22348),
("Bor", 44.07488, 22.09591),
("Borca", 44.86969, 20.45413),
("Bosilegrad", 42.5011, 22.47238),
("Bosut", 44.92977, 19.36086),
("Botos", 45.30837, 20.63514),
("Brasina", 44.4652, 19.15673),
("Brdarica", 44.55376, 19.7715),
("Brezovica", 43.74669, 20.41436),
("Brezovica", 42.95229, 22.17869),
("Buganovci", 44.89388, 19.86344),
("Bujanovac", 42.45917, 21.76667),
("Bukor", 44.49523, 19.57116),
("Bustranje", 42.33152, 21.75407),
("Cacak", 43.89139, 20.34972),
("Cajetina", 43.74977, 19.71273),
("Celarevo", 45.26999, 19.52484),
("Cenej", 45.36826, 19.80423),
("Centa", 45.10814, 20.38947),
("Cestereg", 45.56361, 20.53194),
("Cicevac", 43.71882, 21.44085),
("Coka", 45.9425, 20.14333),
("Cokesina", 44.65319, 19.39016),
("Cortanovci", 45.1546, 20.01851),
("Crepaja", 45.00984, 20.63702),
("Crna Bara", 45.97326, 20.27614),
("Crna Bara", 44.87374, 19.3948),
("Crna Trava", 42.80988, 22.29881),
("Crvenka", 45.66098, 19.45435),
("Cukarica", 44.7825, 20.42028),
("Culjkovic", 44.66595, 19.54562),
("Cuprija", 43.9275, 21.37),
("Curug", 45.47221, 20.06861),
("Debeljaca", 45.0707, 20.60153),
("Despotovac", 44.0925, 21.44694),
("Despotovo", 45.45983, 19.52653),
("Dimitrovgrad", 43.01417, 22.77556),
("Dobanovci", 44.82631, 20.22487),
("Dobric", 44.70224, 19.57931),
("Dobrica", 45.21339, 20.84995),
("Dolac naselje", 43.29511, 22.20626),
("Doljevac", 43.19817, 21.83258),
("Donja Badanja", 44.50556, 19.47543),
("Donja Dubona", 44.52136, 20.7268),
("Donja Gorevnica", 43.87701, 20.50004),
("Donja Konjusa", 43.22293, 21.41421),
("Donji Dobric", 44.61183, 19.33109),
("Donji Milanovac", 44.46593, 22.1517),
("Doroslovo", 45.60699, 19.18868),
("Draginje", 44.53302, 19.7625),
("Drenovac", 44.86649, 19.70943),
("Dublje", 44.80117, 19.50902),
("Duboka", 44.52281, 21.75987),
("Durdevo", 45.32591, 20.06532),
("Durici", 43.84678, 19.40935),
("Ecka", 45.32328, 20.44294),
("Elemir", 45.44263, 20.30003),
("Erdevik", 45.11721, 19.40881),
("Farkazdin", 45.19172, 20.47239),
("Feketic", 45.66431, 19.69974),
("Futog", 45.23874, 19.71416)
ON CONFLICT DO NOTHING;;
