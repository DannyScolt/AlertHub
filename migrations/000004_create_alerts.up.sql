CREATE TYPE alert_severity AS ENUM (
  'info',
  'warning',
  'critical'
);

CREATE TABLE alerts (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  device_id   UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
  client_id   UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
  type        VARCHAR(100) NOT NULL,
  severity    alert_severity NOT NULL,
  message     TEXT NOT NULL,
  payload     JSONB NOT NULL DEFAULT '{}'::jsonb,
  occurred_at TIMESTAMPTZ NOT NULL,
  received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT alerts_type_not_blank CHECK (length(trim(type)) > 0),
  CONSTRAINT alerts_message_not_blank CHECK (length(trim(message)) > 0)
);

CREATE INDEX idx_alerts_client_time ON alerts (client_id, occurred_at DESC);
CREATE INDEX idx_alerts_device_time ON alerts (device_id, occurred_at DESC);
CREATE INDEX idx_alerts_critical ON alerts (client_id, occurred_at DESC) WHERE severity IN ('warning', 'critical');
