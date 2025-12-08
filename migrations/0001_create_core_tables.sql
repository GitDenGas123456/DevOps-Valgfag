-- 0001_create_core_tables.sql
-- Core tables for the application (PostgreSQL schema)

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) NOT NULL UNIQUE,
    email    VARCHAR(255) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL
);

CREATE TABLE IF NOT EXISTS pages (
    id SERIAL PRIMARY KEY,
    title        VARCHAR(255) NOT NULL UNIQUE,
    url          VARCHAR(255) NOT NULL UNIQUE,
    language     VARCHAR(2) NOT NULL
                   CHECK (language IN ('en', 'da'))
                   DEFAULT 'en',

    last_updated TIMESTAMP,
    content      TEXT NOT NULL
);
