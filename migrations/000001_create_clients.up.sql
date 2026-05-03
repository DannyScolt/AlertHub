CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE clients (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email VARCHAR(255) NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  name VARCHAR(100) NOT NULL,
  remember_token TEXT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT clients_email_not_blank CHECK (length(trim(email)) > 0),
  CONSTRAINT clients_name_not_blank CHECK (length(trim(name)) > 0)
);

CREATE INDEX idx_clients_email ON clients (lower(email));
