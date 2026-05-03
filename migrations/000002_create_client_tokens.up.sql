CREATE TABLE client_tokens (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  client_id UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
  name VARCHAR(100) NOT NULL DEFAULT 'auth_session',
  token_hash TEXT NOT NULL UNIQUE,
  token_family UUID NOT NULL,
  abilities JSONB NOT NULL DEFAULT '[]'::jsonb,
  parent_id UUID NULL REFERENCES client_tokens(id),
  replaced_by_id UUID NULL REFERENCES client_tokens(id),
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_used_at TIMESTAMPTZ NULL,
  revoked_at TIMESTAMPTZ NULL,
  revoke_reason TEXT NULL,
  user_agent TEXT NULL,
  ip_address INET NULL,
  CONSTRAINT client_tokens_name_not_blank CHECK (length(trim(name)) > 0),
  CONSTRAINT client_tokens_abilities_array CHECK (jsonb_typeof(abilities) = 'array')
);

CREATE INDEX idx_client_tokens_client_id ON client_tokens(client_id);
CREATE INDEX idx_client_tokens_token_family ON client_tokens(token_family);
CREATE INDEX idx_client_tokens_expires_at ON client_tokens(expires_at);
CREATE INDEX idx_client_tokens_active_client ON client_tokens(client_id, expires_at) WHERE revoked_at IS NULL;
