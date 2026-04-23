DROP INDEX IF EXISTS idx_virtual_machine_hosts_last_seen_at;
DROP INDEX IF EXISTS idx_virtual_machine_hosts_zone_status;
DROP TABLE IF EXISTS virtual_machine.hosts;
DROP SCHEMA IF EXISTS virtual_machine;
