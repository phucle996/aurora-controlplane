CREATE SCHEMA IF NOT EXISTS smtp;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

DO $$ BEGIN
  CREATE TYPE smtp.consumer_transport_type AS ENUM ('redis_stream', 'rabbitmq', 'kafka', 'nats');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
  CREATE TYPE smtp.consumer_status AS ENUM ('active', 'disabled');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
  CREATE TYPE smtp.template_status AS ENUM ('draft', 'review', 'live');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
  CREATE TYPE smtp.gateway_status AS ENUM ('active', 'draining', 'disabled');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
  CREATE TYPE smtp.endpoint_status AS ENUM ('active', 'draining', 'disabled');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
  CREATE TYPE smtp.tls_mode AS ENUM ('none', 'starttls', 'tls', 'mtls');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
