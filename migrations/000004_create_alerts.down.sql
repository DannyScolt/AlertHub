DROP INDEX IF EXISTS idx_alerts_critical;
DROP INDEX IF EXISTS idx_alerts_device_time;
DROP INDEX IF EXISTS idx_alerts_client_time;
DROP TABLE IF EXISTS alerts;
DROP TYPE IF EXISTS alert_severity;
