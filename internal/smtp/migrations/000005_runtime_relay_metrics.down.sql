ALTER TABLE smtp.consumer_runtime_statuses
  DROP COLUMN IF EXISTS relay_queue_depth,
  DROP COLUMN IF EXISTS active_workers,
  DROP COLUMN IF EXISTS desired_workers,
  DROP COLUMN IF EXISTS oldest_unacked_age_ms,
  DROP COLUMN IF EXISTS broker_lag;

ALTER TABLE smtp.gateway_runtime_statuses
  DROP COLUMN IF EXISTS backpressure_state,
  DROP COLUMN IF EXISTS send_rate_per_second,
  DROP COLUMN IF EXISTS pool_busy_conns,
  DROP COLUMN IF EXISTS pool_open_conns,
  DROP COLUMN IF EXISTS relay_queue_depth,
  DROP COLUMN IF EXISTS active_workers,
  DROP COLUMN IF EXISTS desired_workers;

ALTER TABLE smtp.consumer_assignments
  DROP COLUMN IF EXISTS target_gateway_grpc_endpoint,
  DROP COLUMN IF EXISTS target_gateway_data_plane_id,
  DROP COLUMN IF EXISTS target_gateway_shard_id,
  DROP COLUMN IF EXISTS target_gateway_id;
