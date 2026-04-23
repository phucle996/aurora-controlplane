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
  END IF;
END;
$$;

DROP FUNCTION IF EXISTS smtp.cleanup_activity_logs_retention();
DROP TRIGGER IF EXISTS trg_smtp_gateway_endpoints_runtime_version ON smtp.gateway_endpoints;
DROP FUNCTION IF EXISTS smtp.touch_gateway_runtime_version();
DROP TRIGGER IF EXISTS trg_smtp_endpoint_secrets_runtime_version ON smtp.endpoint_secrets;
DROP FUNCTION IF EXISTS smtp.touch_endpoint_runtime_version();
DROP TRIGGER IF EXISTS trg_smtp_consumer_secrets_runtime_version ON smtp.consumer_secrets;
DROP FUNCTION IF EXISTS smtp.touch_consumer_runtime_version();
DROP TRIGGER IF EXISTS trg_smtp_endpoints_runtime_version ON smtp.endpoints;
DROP TRIGGER IF EXISTS trg_smtp_gateways_runtime_version ON smtp.gateways;
DROP TRIGGER IF EXISTS trg_smtp_templates_runtime_version ON smtp.templates;
DROP TRIGGER IF EXISTS trg_smtp_consumers_runtime_version ON smtp.consumers;
DROP FUNCTION IF EXISTS smtp.assign_runtime_version();
