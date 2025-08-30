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
INSERT INTO cities (name, country_id, latitude, longitude) VALUES
('Ada', 36, 45.8025, 20.12583), ('Adorjan', 36, 46.00333, 20.04007), ('Aleksandrovac', 36, 43.45861, 21.0525), ('Aleksandrovo', 36, 45.63755, 20.59288), ('Aleksinac', 36, 43.54167, 21.70778), ('Alibunar', 36, 45.08083, 20.96583), ('Apatin', 36, 45.6726, 18.978), ('Aradac', 36, 45.38346, 20.30137), ('Arangelovac', 36, 44.30694, 20.56), ('Arilje', 36, 43.75306, 20.09556), ('Bac', 36, 45.39194, 19.23667), ('Bace', 36, 43.21963, 21.34463), ('Backa Palanka', 36, 45.24966, 19.39664), ('Backa Topola', 36, 45.81516, 19.6318), ('Backi Breg', 36, 45.92034, 18.92944), ('Backi Petrovac', 36, 45.36056, 19.59167), ('Backo Gradiste', 36, 45.53271, 20.03082), ('Backo Petrovo Selo', 36, 45.70681, 20.07928), ('Badovinci', 36, 44.78534, 19.37146), ('Bajina Basta', 36, 43.97083, 19.5675), ('Banatska Dubica', 36, 45.27146, 20.82833), ('Banatska Topola', 36, 45.67248, 20.4653), ('Banatski Despotovac', 36, 45.36606, 20.66407), ('Banatski Dvor', 36, 45.51866, 20.51146), ('Banatski Karlovac', 36, 45.04987, 21.018), ('Banatsko Karagorgevo', 36, 45.58693, 20.56421), ('Banatsko Veliko Selo', 36, 45.81961, 20.60772), ('Banovo Polje', 36, 44.9104, 19.44916), ('Barajevo', 36, 44.57889, 20.41583), ('Baranda', 36, 45.08459, 20.44264), ('Baric', 36, 44.6507, 20.25941), ('Barice', 36, 45.18189, 21.08279), ('Basaid', 36, 45.64102, 20.41434), ('Batocina', 36, 44.15361, 21.08167), ('Bavaniste', 36, 44.81907, 20.87654), ('Becej', 36, 45.61632, 20.03331), ('Becmen', 36, 44.77983, 20.20577), ('Bela Crkva', 36, 44.8975, 21.41722), ('Bela Palanka', 36, 43.21833, 22.31111), ('Belegis', 36, 45.0192, 20.33323), ('Belgrade', 36, 44.80401, 20.46513), ('Belo Blato', 36, 45.27278, 20.375), ('Beloljin', 36, 43.23846, 21.39673), ('Belotic', 36, 44.81782, 19.54801), ('Belotic', 36, 44.58099, 19.71932), ('Beocin', 36, 45.20829, 19.72063), ('Besenovo', 36, 45.08364, 19.70408), ('Beska', 36, 45.13092, 20.06698), ('Biljaca', 36, 42.3543, 21.7437), ('Blace', 36, 43.29528, 21.28583), ('Bocar', 36, 45.76994, 20.2839), ('Bogatic', 36, 44.8375, 19.48056), ('Bogojevo', 36, 45.53015, 19.13022), ('Bogosavac', 36, 44.71799, 19.59533), ('Bojnik', 36, 43.01224, 21.721), ('Boka', 36, 45.3554, 20.82987), ('Boljevac', 36, 43.83028, 21.95306), ('Boljevci', 36, 44.72355, 20.22348), ('Bor', 36, 44.07488, 22.09591), ('Borca', 36, 44.86969, 20.45413), ('Bosilegrad', 36, 42.5011, 22.47238), ('Bosut', 36, 44.92977, 19.36086), ('Botos', 36, 45.30837, 20.63514), ('Brasina', 36, 44.4652, 19.15673), ('Brdarica', 36, 44.55376, 19.7715), ('Brezovica', 36, 43.74669, 20.41436), ('Brezovica', 36, 42.95229, 22.17869), ('Buganovci', 36, 44.89388, 19.86344), ('Bujanovac', 36, 42.45917, 21.76667), ('Bukor', 36, 44.49523, 19.57116), ('Bustranje', 36, 42.33152, 21.75407), ('Cacak', 36, 43.89139, 20.34972), ('Cajetina', 36, 43.74977, 19.71273), ('Celarevo', 36, 45.26999, 19.52484), ('Cenej', 36, 45.36826, 19.80423), ('Centa', 36, 45.10814, 20.38947), ('Cestereg', 36, 45.56361, 20.53194), ('Cicevac', 36, 43.71882, 21.44085), ('Coka', 36, 45.9425, 20.14333), ('Cokesina', 36, 44.65319, 19.39016), ('Cortanovci', 36, 45.1546, 20.01851), ('Crepaja', 36, 45.00984, 20.63702), ('Crna Bara', 36, 45.97326, 20.27614), ('Crna Bara', 36, 44.87374, 19.3948), ('Crna Trava', 36, 42.80988, 22.29881), ('Crvenka', 36, 45.66098, 19.45435), ('Cukarica', 36, 44.7825, 20.42028), ('Culjkovic', 36, 44.66595, 19.54562), ('Cuprija', 36, 43.9275, 21.37), ('Curug', 36, 45.47221, 20.06861), ('Debeljaca', 36, 45.0707, 20.60153), ('Despotovac', 36, 44.0925, 21.44694), ('Despotovo', 36, 45.45983, 19.52653), ('Dimitrovgrad', 36, 43.01417, 22.77556), ('Dobanovci', 36, 44.82631, 20.22487), ('Dobric', 36, 44.70224, 19.57931), ('Dobrica', 36, 45.21339, 20.84995), ('Dolac naselje', 36, 43.29511, 22.20626), ('Doljevac', 36, 43.19817, 21.83258), ('Donja Badanja', 36, 44.50556, 19.47543), ('Donja Dubona', 36, 44.52136, 20.7268), ('Donja Gorevnica', 36, 43.87701, 20.50004), ('Donja Konjusa', 36, 43.22293, 21.41421), ('Donji Dobric', 36, 44.61183, 19.33109), ('Donji Milanovac', 36, 44.46593, 22.1517), ('Doroslovo', 36, 45.60699, 19.18868), ('Draginje', 36, 44.53302, 19.7625), ('Drenovac', 36, 44.86649, 19.70943), ('Dublje', 36, 44.80117, 19.50902), ('Duboka', 36, 44.52281, 21.75987), ('Durdevo', 36, 45.32591, 20.06532), ('Durici', 36, 43.84678, 19.40935), ('Ecka', 36, 45.32328, 20.44294), ('Elemir', 36, 45.44263, 20.30003), ('Erdevik', 36, 45.11721, 19.40881), ('Farkazdin', 36, 45.19172, 20.47239), ('Feketic', 36, 45.66431, 19.69974), ('Futog', 36, 45.23874, 19.71416), ('Gadzin Han', 36, 43.22278, 22.03194), ('Gakovo', 36, 45.90078, 19.06138), ('Gamzigrad', 36, 43.90873, 22.17507), ('Gardinovci', 36, 45.20359, 20.13558), ('Glogovac', 36, 44.04213, 21.3134), ('Glogovac', 36, 44.86341, 19.41532), ('Glozan', 36, 45.27954, 19.56838), ('Glusci', 36, 44.89021, 19.54913), ('Golubac', 36, 44.65296, 21.63199), ('Golubinci', 36, 44.98533, 20.06339), ('Gornja Bukovica', 36, 44.34233, 19.78647), ('Gornje Nedeljice', 36, 44.51645, 19.33915), ('Gornji Breg', 36, 45.91995, 20.01766), ('Gornji Dobric', 36, 44.58286, 19.30858), ('Gornji Milanovac', 36, 44.02603, 20.46152), ('Grabovac', 36, 44.60049, 20.08539), ('Grabovci', 36, 44.76496, 19.84489), ('Grncara', 36, 44.53516, 19.29971), ('Grocka', 36, 44.67152, 20.71648), ('Gudurica', 36, 45.16816, 21.44264), ('Hajducica', 36, 45.2501, 20.96016), ('Hetin', 36, 45.66202, 20.79138), ('Hrtkovci', 36, 44.88155, 19.76374), ('Idos', 36, 45.82648, 20.31791), ('Idvor', 36, 45.18895, 20.51442), ('Ilandza', 36, 45.16897, 20.92008), ('Ingija', 36, 45.04816, 20.08165), ('Irig', 36, 45.10111, 19.85833), ('Irig', 36, 45.0523, 19.84448), ('Ivanjica', 36, 43.58028, 20.23111), ('Izbiste', 36, 45.02253, 21.18388), ('Jablanka', 36, 45.07524, 21.39067), ('Jadranska Lesnica', 36, 44.58625, 19.34701), ('Jagodina', 36, 43.97713, 21.26121), ('Jajinci', 36, 44.74061, 20.48634), ('Jakovo', 36, 44.75366, 20.26008), ('Jankov Most', 36, 45.47498, 20.43835), ('Janosik', 36, 45.17141, 21.00658), ('Jarak', 36, 44.91843, 19.75477), ('Jarebice', 36, 44.53995, 19.42418), ('Jarkovac', 36, 45.26985, 20.76078), ('Jasa Tomic', 36, 45.44725, 20.85546), ('Jazovo', 36, 45.89876, 20.2213), ('Jelenca', 36, 44.727, 19.735), ('Jermenovci', 36, 45.18635, 21.0455), ('Jevremovac', 36, 44.72172, 19.66364), ('Joseva', 36, 44.58772, 19.40967), ('Kac', 36, 45.30407, 19.94299), ('Kamenica', 36, 44.343, 19.72333), ('Kanjiza', 36, 46.06667, 20.05), ('Kikinda', 36, 45.82972, 20.46528), ('Kisac', 36, 45.35421, 19.72975), ('Kladovo', 36, 44.61147, 22.60955), ('Klek', 36, 45.42254, 20.48049), ('Klenak', 36, 44.78846, 19.71004), ('Klenje', 36, 44.80794, 19.43508), ('Knic', 36, 43.92694, 20.71889), ('Knicanin', 36, 45.18675, 20.319), ('Knjazevac', 36, 43.56634, 22.25701), ('Koceljeva', 36, 44.47361, 19.81167), ('Kolut', 36, 45.89292, 18.9276), ('Konak', 36, 45.31575, 20.91468), ('Kosjeric', 36, 43.99611, 19.90694), ('Kovacica', 36, 45.11167, 20.62139), ('Kovilj', 36, 45.23422, 20.02327), ('Kovilovo', 36, 44.90791, 20.42319), ('Kovin', 36, 44.7475, 20.97611), ('Kozjak', 36, 45.18264, 20.86381), ('Kozjak', 36, 44.58727, 19.28412), ('Kragujevac', 36, 44.01667, 20.91667), ('Krajisnik', 36, 45.45283, 20.72976), ('Kraljevo', 36, 43.72583, 20.68944), ('Krcedin', 36, 45.13871, 20.13308), ('Kremna', 36, 43.83807, 19.57433), ('Kriva Feja', 36, 42.55909, 22.17487), ('Krivaja', 36, 44.55021, 19.59153), ('Krupanj', 36, 44.36556, 19.36194), ('Krusevac', 36, 43.58, 21.33389), ('Kucevo', 36, 44.4775, 21.67), ('Kula', 36, 45.60889, 19.52639), ('Kulpin', 36, 45.4024, 19.58814), ('Kumane', 36, 45.53946, 20.22902), ('Kupinik', 36, 45.29254, 21.13702), ('Kupinovo', 36, 44.70708, 20.04959), ('Kupusina', 36, 45.73759, 19.01082), ('Kursumlija', 36, 43.13826, 21.27339), ('Kustilj', 36, 45.03487, 21.37989), ('Lagja e Korbajve', 36, 42.38674, 21.7434), ('Lagja e Poshtme', 36, 42.38853, 21.72971), ('Lagja e Shimshirve', 36, 42.38025, 21.74045), ('Lagja e Ternovcalive', 36, 42.38999, 21.74365), ('Lajkovac', 36, 44.36944, 20.16528), ('Lapovo', 36, 44.18424, 21.09727), ('Lazarevac', 36, 44.38534, 20.2557), ('Lazarevo', 36, 45.38893, 20.53999), ('Lebane', 36, 42.92389, 21.7375), ('Ledinci', 36, 45.18697, 19.80554), ('Leskovac', 36, 42.99806, 21.94611), ('Lesnica', 36, 44.6525, 19.31), ('Lipnicki Sor', 36, 44.58058, 19.26572), ('Lipolist', 36, 44.69783, 19.50101), ('Ljig', 36, 44.23007, 20.23819), ('Ljubovija', 36, 44.18944, 19.37667), ('Ljukovo', 36, 45.02604, 20.02737), ('Lok', 36, 45.21583, 20.21222), ('Lokve', 36, 45.15198, 21.03073), ('Lovcenac', 36, 45.6816, 19.69205), ('Loznica', 36, 44.53118, 19.24267), ('Lucani', 36, 43.86083, 20.13806), ('Lugavcina', 36, 44.52314, 21.07083), ('Lukicevo', 36, 45.33815, 20.49895), ('Lukino Selo', 36, 45.30244, 20.42632), ('Macvanska Mitrovica', 36, 44.96739, 19.59314), ('Maglic', 36, 45.36248, 19.53211), ('Majdanpek', 36, 44.42771, 21.94596), ('Majur', 36, 44.77105, 19.65512), ('Mala Ivanca', 36, 44.58961, 20.61634), ('Mala Mostanica', 36, 44.63834, 20.306), ('Mali Igos', 36, 45.70833, 19.66528), ('Mali Pozarevac', 36, 44.56327, 20.65433), ('Mali Zam', 36, 45.20979, 21.33729), ('Mali Zvornik', 36, 44.37344, 19.10651), ('Malo Crnice', 36, 44.55611, 21.28556), ('Margita', 36, 45.21598, 21.17527), ('Markovac', 36, 45.14964, 21.47242), ('Medvega', 36, 42.84306, 21.58333), ('Mega', 36, 45.53815, 20.80677), ('Melenci', 36, 45.5168, 20.31961), ('Merosina', 36, 43.28358, 21.7204), ('Metkovic', 36, 44.85617, 19.54654), ('Mihajlovo', 36, 45.47085, 20.41508), ('Mileticevo', 36, 45.30508, 21.06404), ('Mionica', 36, 44.25194, 20.08167), ('Miratovac', 36, 42.25846, 21.66456), ('Mladenovac', 36, 44.43861, 20.69917), ('Mokra Gora', 36, 43.78853, 19.50033), ('Mokrin', 36, 45.93362, 20.41215), ('Mol', 36, 45.76457, 20.13286), ('Mosorin', 36, 45.30196, 20.16919), ('Mrovska', 36, 44.54278, 19.675), ('Nakovo', 36, 45.87503, 20.56709), ('Nakucani', 36, 44.6014, 19.66718), ('Negotin', 36, 44.22639, 22.53083), ('Neuzina', 36, 45.3446, 20.71418), ('Nikinci', 36, 44.85017, 19.82321), ('Nikolinci', 36, 45.05245, 21.06695), ('Nis', 36, 43.32472, 21.90333), ('Niska Banja', 36, 43.29507, 22.0057), ('Nova Crnja', 36, 45.66833, 20.605), ('Nova Pazova', 36, 44.94366, 20.21931), ('Nova Varos', 36, 43.46056, 19.81139), ('Novi Banovci', 36, 44.95691, 20.28076), ('Novi Becej', 36, 45.59861, 20.13556), ('Novi Beograd', 36, 44.80556, 20.42417), ('Novi Itebej', 36, 45.55918, 20.7003), ('Novi Karlovci', 36, 45.07636, 20.17948), ('Novi Knezevac', 36, 46.05, 20.1), ('Novi Kozarci', 36, 45.78241, 20.62289), ('Novi Pazar', 36, 43.13667, 20.51222), ('Novi Sad', 36, 45.25167, 19.83694), ('Novi Slankamen', 36, 45.12554, 20.23914), ('Novi Zednik', 36, 45.94117, 19.67141), ('Novo Milosevo', 36, 45.71916, 20.30364), ('Novo Selo', 36, 44.67041, 19.34495), ('Obrenovac', 36, 44.65486, 20.20017), ('Obrovac', 36, 45.32106, 19.35048), ('Odzaci', 36, 45.50667, 19.26111), ('Opovo', 36, 45.05222, 20.43028), ('Orlovat', 36, 45.24171, 20.58089), ('Osecina', 36, 44.37306, 19.60139), ('Osjecenik', 36, 43.14528, 19.85889), ('Ostojicevo', 36, 45.88863, 20.16642), ('Ostruznica', 36, 44.72769, 20.31845), ('Ovca', 36, 44.88349, 20.53336), ('Padej', 36, 45.82756, 20.16279), ('Padina', 36, 45.11988, 20.7286), ('Padinska Skela', 36, 44.94291, 20.42549), ('Palic', 36, 46.10646, 19.75647), ('Palilula', 36, 44.81167, 20.51611), ('Pancevo', 36, 44.87177, 20.64167), ('Paracin', 36, 43.86083, 21.40778), ('Pavlis', 36, 45.10569, 21.23952), ('Pecinci', 36, 44.90889, 19.96639), ('Perlez', 36, 45.20813, 20.38197), ('Petkovica', 36, 44.66627, 19.43923), ('Petrovac', 36, 44.37694, 21.41917), ('Petrovaradin', 36, 45.24667, 19.87944), ('Pirot', 36, 43.15306, 22.58611), ('Pirot Centralni Trg', 36, 43.15635, 22.58515), ('Plandiste', 36, 45.22722, 21.12167), ('Platicevo', 36, 44.82213, 19.79487), ('Pocerski Pricinovic', 36, 44.72222, 19.70722), ('Pozarevac', 36, 44.62133, 21.18782), ('Pozega', 36, 43.84889, 20.03632), ('Preljina', 36, 43.91454, 20.40677), ('Presevo', 36, 42.30917, 21.64917), ('Priboj', 36, 43.58306, 19.52519), ('Prigrevica', 36, 45.67636, 19.08809), ('Prijepolje', 36, 43.38996, 19.6487), ('Prislonica', 36, 43.95223, 20.43521), ('Prnjavor', 36, 44.70061, 19.38695), ('Progar', 36, 44.71744, 20.16047), ('Prokuplje', 36, 43.23417, 21.58806), ('Putinci', 36, 44.99259, 19.97102), ('Raca', 36, 44.22712, 20.97754), ('Radenka', 36, 44.58345, 21.76469), ('Radenkovic', 36, 44.92191, 19.49543), ('Radojevo', 36, 45.74617, 20.78917), ('Radovnica', 36, 42.43364, 22.22861), ('Rajince', 36, 42.3787, 21.69591), ('Rakovica', 36, 44.74194, 20.44139), ('Ralja', 36, 44.57222, 20.56222), ('Raska', 36, 43.28722, 20.61528), ('Ratkovo', 36, 45.45566, 19.33728), ('Ravanica', 36, 43.74771, 20.8843), ('Ravni Topolovac', 36, 45.46082, 20.56939), ('Ravnje', 36, 44.94326, 19.4228), ('Ravno Selo', 36, 45.44967, 19.62097), ('Razanj', 36, 43.67222, 21.54944), ('Rekovac', 36, 43.863, 21.09345), ('Ribari', 36, 44.70961, 19.42472), ('Rigica', 36, 45.99088, 19.10635), ('Ripanj', 36, 44.63864, 20.52136), ('Ritisevo', 36, 45.06454, 21.2281), ('Ritopek', 36, 44.73849, 20.65499), ('Rudnik', 36, 44.13982, 20.49398), ('Ruma', 36, 45.00806, 19.82222), ('Rumenka', 36, 45.294, 19.74306), ('Rumska', 36, 44.57261, 19.58988), ('Rusanj', 36, 44.68477, 20.44993), ('Ruski Krstur', 36, 45.56392, 19.42002), ('Rusko Selo', 36, 45.76291, 20.57117), ('Sabac', 36, 44.74667, 19.69), ('Sajan', 36, 45.84227, 20.27815), ('Sajkas', 36, 45.27315, 20.09051), ('Sakule', 36, 45.14667, 20.48619), ('Salas Crnobarski', 36, 44.82843, 19.39437), ('Salas Nocajski', 36, 44.94722, 19.58611), ('Samoljica', 36, 42.38445, 21.73708), ('Samos', 36, 45.20255, 20.77392), ('Sanad', 36, 45.97596, 20.10816), ('Sasinci', 36, 44.96514, 19.74151), ('Savski Venac', 36, 44.77917, 20.45389), ('Secanj', 36, 45.36667, 20.77222), ('Sefkerin', 36, 45.00501, 20.48256), ('Seleus', 36, 45.1277, 20.91461), ('Senta', 36, 45.9275, 20.07722), ('Sevarice', 36, 44.86704, 19.66006), ('Sevica', 36, 44.50883, 21.72296), ('Sid', 36, 45.12833, 19.22639), ('Simanovci', 36, 44.87393, 20.09175), ('Sinosevic', 36, 44.61503, 19.63601), ('Sirogojno', 36, 43.689, 19.88746), ('Sjenica', 36, 43.27306, 19.99944), ('Smederevo', 36, 44.66436, 20.92763), ('Smederevska Palanka', 36, 44.36548, 20.95885), ('Soko Banja', 36, 43.64333, 21.87111), ('Sokolovica', 36, 43.21528, 20.31556), ('Sokolovo Brdo', 36, 43.13694, 19.80556), ('Sombor', 36, 45.77417, 19.11222), ('Sonta', 36, 45.59427, 19.09719), ('Sopot', 36, 44.51972, 20.57361), ('Sot', 36, 45.16077, 19.31782), ('Spancevac', 36, 42.36263, 21.85619), ('Srbobran', 36, 45.55389, 19.80278), ('Sremcica', 36, 44.67653, 20.39232), ('Sremska Kamenica', 36, 45.22334, 19.84263), ('Sremska Mitrovica', 36, 44.97639, 19.61222), ('Sremska Raca', 36, 44.92077, 19.28577), ('Sremski Karlovci', 36, 45.20285, 19.93373), ('Srpska Crnja', 36, 45.72538, 20.69008), ('Srpski Itebej', 36, 45.56715, 20.7135), ('Stajicevo', 36, 45.29489, 20.45845), ('Stanisic', 36, 45.93895, 19.16709), ('Stara Pazova', 36, 44.985, 20.16083), ('Starcevo', 36, 44.80729, 20.70546), ('Stari Banovci', 36, 44.9842, 20.28382), ('Stari Grad', 36, 44.81789, 20.46186), ('Stari Lec', 36, 45.28401, 20.96433), ('Stari Slankamen', 36, 45.14249, 20.25765), ('Stari Zednik', 36, 45.94864, 19.63029), ('Stepanovicevo', 36, 45.41369, 19.7), ('Stepojevac', 36, 44.51278, 20.295), ('Stitar', 36, 44.79415, 19.59529), ('Stubline', 36, 44.57476, 20.13477), ('Subotica', 36, 46.1, 19.66667), ('Sumulice', 36, 42.38682, 21.734), ('Surcin', 36, 44.79306, 20.28028), ('Surduk', 36, 45.07118, 20.3251), ('Surdulica', 36, 42.69056, 22.17083), ('Sutjeska', 36, 45.38312, 20.6962), ('Suvi Do', 36, 43.25829, 21.33269), ('Svilajnac', 36, 44.2337, 21.1967), ('Svrljig', 36, 43.41333, 22.12111), ('Tabanovic', 36, 44.82018, 19.64128), ('Taras', 36, 45.46737, 20.19867), ('Tekeris', 36, 44.55726, 19.5297), ('Temerin', 36, 45.40861, 19.88917), ('Titel', 36, 45.20611, 20.29444), ('Toba', 36, 45.68943, 20.55714), ('Tomasevac', 36, 45.26855, 20.62272), ('Topola', 36, 44.25417, 20.6825), ('Torak', 36, 45.50928, 20.609), ('Torda', 36, 45.58423, 20.459), ('Trgoviste', 36, 42.36417, 22.0825), ('Trsic', 36, 44.49502, 19.2649), ('Trstenik', 36, 43.61694, 21.0025), ('Tulare', 36, 43.228, 21.37961), ('Turija', 36, 44.52273, 21.63945), ('Tutin', 36, 42.99028, 20.33139), ('Ub', 36, 44.45611, 20.07389), ('Ugrinovci', 36, 44.87635, 20.18763), ('Uljma', 36, 45.04213, 21.15393), ('Umka', 36, 44.67806, 20.30472), ('Uzdin', 36, 45.20512, 20.62342), ('Uzice', 36, 43.85861, 19.84878), ('Uzvece', 36, 44.87861, 19.60356), ('Valjevo', 36, 44.27513, 19.89821), ('Varna', 36, 44.67914, 19.6515), ('Varvarin', 36, 43.72397, 21.3624), ('Velika Greda', 36, 45.24376, 21.03498), ('Velika Ivanca', 36, 44.42639, 20.58028), ('Velika Mostanica', 36, 44.66486, 20.35395), ('Velika Plana', 36, 44.33389, 21.07676), ('Veliki Gaj', 36, 45.28849, 21.17057), ('Veliko Gradiste', 36, 44.76329, 21.51646), ('Veliko Srediste', 36, 45.17919, 21.40353), ('Veternik', 36, 45.25446, 19.7588), ('Vilovo', 36, 45.24859, 20.15521), ('Vinca', 36, 44.75907, 20.61747), ('Visnjicevo', 36, 44.96731, 19.28993), ('Vladicin Han', 36, 42.70778, 22.06333), ('Vladimirci', 36, 44.61472, 19.78528), ('Vladimirovac', 36, 45.03122, 20.86566), ('Vlajkovac', 36, 45.07207, 21.19945), ('Vlasotince', 36, 42.96697, 22.13402), ('Vojka', 36, 44.93713, 20.15236), ('Vojvoda Stepa', 36, 45.68537, 20.65536), ('Vojvodinci', 36, 45.01557, 21.34793), ('Vozdovac', 36, 44.77833, 20.47583), ('Vracar', 36, 44.79256, 20.47491), ('Vranic', 36, 44.60237, 20.32872), ('Vranje', 36, 42.55139, 21.90028), ('Vranjska Banja', 36, 42.555, 21.99222), ('Vrbas', 36, 45.57139, 19.64083), ('Vrbica', 36, 46.00584, 20.31325), ('Vrcin', 36, 44.66845, 20.59325), ('Vrdnik', 36, 45.12174, 19.79227), ('Vrnjacka Banja', 36, 43.62725, 20.89634), ('Vrsac', 36, 45.11667, 21.30361), ('Zabalj', 36, 45.37222, 20.06389), ('Zabari', 36, 44.35611, 21.215), ('Zagubica', 36, 44.19685, 21.78838), ('Zajecar', 36, 43.90358, 22.26405), ('Zasavica Prva', 36, 44.95448, 19.48938), ('Zemun', 36, 44.8458, 20.40116), ('Zitiste', 36, 45.485, 20.54972), ('Zitni Potok', 36, 43.09021, 21.57476), ('Zitoraga', 36, 43.19, 21.71306), ('Zlatibor', 36, 43.729, 19.70029), ('Zmajevo', 36, 45.45408, 19.6905), ('Zminjak', 36, 44.75711, 19.4707), ('Zrenjanin', 36, 45.38361, 20.38194), ('Zuce', 36, 44.69806, 20.55524), ('Zujince', 36, 42.31568, 21.70212), ('Zvecka', 36, 44.64025, 20.16432), ('Zvezdara', 36, 44.77465, 20.53207)
ON CONFLICT DO NOTHING;
