CREATE TABLE IF NOT EXISTS todos (
        id SERIAL PRIMARY KEY,
        todo TEXT,
        created_at TIMESTAMP WITHOUT TIME ZONE,
        updated_at TIMESTAMP WITHOUT TIME ZONE
);

CREATE TABLE IF NOT EXISTS secrets (
        id SERIAL PRIMARY KEY,
        key TEXT
);
