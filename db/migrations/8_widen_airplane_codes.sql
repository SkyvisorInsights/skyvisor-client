-- Aviationstack occasionally returns code fields longer than the original
-- varchar limits (observed: >20 chars in airline code fields). Widen to text;
-- these are display fields with no length-dependent index semantics.
ALTER TABLE airplane
    ALTER COLUMN airline_iata_code TYPE text,
    ALTER COLUMN iata_code_long TYPE text,
    ALTER COLUMN iata_code_short TYPE text,
    ALTER COLUMN airline_icao_code TYPE text,
    ALTER COLUMN engines_type TYPE text,
    ALTER COLUMN icao_code_hex TYPE text,
    ALTER COLUMN line_number TYPE text,
    ALTER COLUMN model_code TYPE text,
    ALTER COLUMN registration_number TYPE text,
    ALTER COLUMN test_registration_number TYPE text;
