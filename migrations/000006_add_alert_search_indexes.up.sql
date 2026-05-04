CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX idx_alerts_message_trgm ON alerts USING gin (message gin_trgm_ops);
CREATE INDEX idx_alerts_type_trgm ON alerts USING gin (type gin_trgm_ops);
CREATE INDEX idx_devices_name_trgm ON devices USING gin (name gin_trgm_ops);
