CREATE TYPE device_status AS ENUM (
  'active',
  'inactive',
  'maintenance',
  'error'
);

CREATE TYPE device_type AS ENUM (
  'temperature_sensor',
  'humidity_sensor',
  'smoke_detector',
  'motion_sensor',
  'door_sensor',
  'camera',
  'gateway',
  'other'
);

CREATE TABLE devices (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  client_id UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
  name VARCHAR(100) NOT NULL,
  type device_type NOT NULL,
  status device_status NOT NULL DEFAULT 'active',
  api_key_hash TEXT NOT NULL UNIQUE,
  tags TEXT[] NOT NULL DEFAULT '{}',
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  last_seen_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ NULL,
  CONSTRAINT devices_name_not_blank CHECK (length(trim(name)) > 0)
);

CREATE INDEX idx_devices_client_id ON devices(client_id);
CREATE INDEX idx_devices_client_status ON devices(client_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_devices_client_type ON devices(client_id, type) WHERE deleted_at IS NULL;
CREATE INDEX idx_devices_deleted_at ON devices(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX idx_devices_api_key_hash ON devices(api_key_hash);
CREATE UNIQUE INDEX idx_devices_client_name_active_unique ON devices(client_id, lower(name)) WHERE deleted_at IS NULL;
