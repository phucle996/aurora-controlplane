CREATE OR REPLACE FUNCTION smtp.assign_runtime_version()
RETURNS trigger AS $$
BEGIN
  NEW.runtime_version := nextval('smtp.runtime_version_seq');
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_smtp_consumers_runtime_version ON smtp.consumers;
CREATE TRIGGER trg_smtp_consumers_runtime_version
BEFORE UPDATE ON smtp.consumers
FOR EACH ROW
EXECUTE FUNCTION smtp.assign_runtime_version();

DROP TRIGGER IF EXISTS trg_smtp_templates_runtime_version ON smtp.templates;
CREATE TRIGGER trg_smtp_templates_runtime_version
BEFORE UPDATE ON smtp.templates
FOR EACH ROW
EXECUTE FUNCTION smtp.assign_runtime_version();

DROP TRIGGER IF EXISTS trg_smtp_gateways_runtime_version ON smtp.gateways;
CREATE TRIGGER trg_smtp_gateways_runtime_version
BEFORE UPDATE ON smtp.gateways
FOR EACH ROW
EXECUTE FUNCTION smtp.assign_runtime_version();

DROP TRIGGER IF EXISTS trg_smtp_endpoints_runtime_version ON smtp.endpoints;
CREATE TRIGGER trg_smtp_endpoints_runtime_version
BEFORE UPDATE ON smtp.endpoints
FOR EACH ROW
EXECUTE FUNCTION smtp.assign_runtime_version();

CREATE OR REPLACE FUNCTION smtp.touch_consumer_runtime_version()
RETURNS trigger AS $$
BEGIN
  UPDATE smtp.consumers
  SET updated_at = now()
  WHERE id = COALESCE(NEW.consumer_id, OLD.consumer_id);
  RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_smtp_consumer_secrets_runtime_version ON smtp.consumer_secrets;
CREATE TRIGGER trg_smtp_consumer_secrets_runtime_version
AFTER INSERT OR UPDATE OR DELETE ON smtp.consumer_secrets
FOR EACH ROW
EXECUTE FUNCTION smtp.touch_consumer_runtime_version();

CREATE OR REPLACE FUNCTION smtp.touch_endpoint_runtime_version()
RETURNS trigger AS $$
BEGIN
  UPDATE smtp.endpoints
  SET updated_at = now()
  WHERE id = COALESCE(NEW.endpoint_id, OLD.endpoint_id);
  RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_smtp_endpoint_secrets_runtime_version ON smtp.endpoint_secrets;
CREATE TRIGGER trg_smtp_endpoint_secrets_runtime_version
AFTER INSERT OR UPDATE OR DELETE ON smtp.endpoint_secrets
FOR EACH ROW
EXECUTE FUNCTION smtp.touch_endpoint_runtime_version();

CREATE OR REPLACE FUNCTION smtp.touch_gateway_runtime_version()
RETURNS trigger AS $$
BEGIN
  UPDATE smtp.gateways
  SET updated_at = now()
  WHERE id = COALESCE(NEW.gateway_id, OLD.gateway_id);
  RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_smtp_gateway_endpoints_runtime_version ON smtp.gateway_endpoints;
CREATE TRIGGER trg_smtp_gateway_endpoints_runtime_version
AFTER INSERT OR UPDATE OR DELETE ON smtp.gateway_endpoints
FOR EACH ROW
EXECUTE FUNCTION smtp.touch_gateway_runtime_version();

CREATE OR REPLACE FUNCTION smtp.cleanup_activity_logs_retention()
RETURNS void
LANGUAGE plpgsql
AS $$
BEGIN
  DELETE FROM smtp.activity_logs
  WHERE created_at < now() - interval '30 day';
END;
$$;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_available_extensions WHERE name = 'pg_cron') THEN
    BEGIN
      CREATE EXTENSION IF NOT EXISTS pg_cron;
    EXCEPTION
      WHEN insufficient_privilege OR undefined_file THEN
        NULL;
    END;
  END IF;
END;
$$;

DO $$
DECLARE
  job_name constant text := 'smtp-activity-logs-retention-daily';
  job_id bigint;
BEGIN
  IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pg_cron') THEN
    SELECT jobid INTO job_id
    FROM cron.job
    WHERE jobname = job_name;

    IF job_id IS NOT NULL THEN
      PERFORM cron.unschedule(job_id);
    END IF;

    PERFORM cron.schedule(
      job_name,
      '0 0 * * *',
      $job$SELECT smtp.cleanup_activity_logs_retention();$job$
    );
  END IF;
END;
$$;
