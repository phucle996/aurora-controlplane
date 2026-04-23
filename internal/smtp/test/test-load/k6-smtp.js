import http from "k6/http";
import exec from "k6/execution";
import { sleep, fail } from "k6";
import { Counter, Trend } from "k6/metrics";
import { b64encode } from "k6/encoding";
import { crypto } from "k6/experimental/webcrypto";

const BASE_URL = trimSlash(__ENV.BASE_URL || "http://127.0.0.1:8080");
const TEST_USERNAME = trim(__ENV.TEST_USERNAME || "smtp.loadtest");
const TEST_PASSWORD = __ENV.TEST_PASSWORD || "Loadtest123!";
const PHASE = trim(__ENV.PHASE || "load").toLowerCase();

const WORKSPACE_ID_OVERRIDE = trim(__ENV.WORKSPACE_ID || "");
const ZONE_ID_OVERRIDE = trim(__ENV.ZONE_ID || "");
const SMTP_MOCK_HOST = trim(__ENV.SMTP_MOCK_HOST || "127.0.0.1");
const SMTP_MOCK_PORT = parseInt(__ENV.SMTP_MOCK_PORT || "2525", 10);

const LOAD_VUS = parseInt(__ENV.LOAD_VUS || "100", 10);
const LOAD_DURATION = __ENV.LOAD_DURATION || "5m";

const STRESS_START_VUS = parseInt(__ENV.STRESS_START_VUS || "10", 10);
const STRESS_STEP_VUS = parseInt(__ENV.STRESS_STEP_VUS || "100", 10);
const STRESS_MAX_VUS = parseInt(__ENV.STRESS_MAX_VUS || "400", 10);
const STRESS_STEP_DURATION = __ENV.STRESS_STEP_DURATION || "2m";

const SPIKE_VUS = parseInt(__ENV.SPIKE_VUS || "800", 10);
const SPIKE_RAMP_UP = __ENV.SPIKE_RAMP_UP || "30s";
const SPIKE_HOLD = __ENV.SPIKE_HOLD || "90s";
const SPIKE_RAMP_DOWN = __ENV.SPIKE_RAMP_DOWN || "30s";

const SOAK_VUS = parseInt(__ENV.SOAK_VUS || "50", 10);
const SOAK_DURATION = __ENV.SOAK_DURATION || "30m";

const REFRESH_INTERVAL_MS = parseInt(__ENV.REFRESH_INTERVAL_MS || "420000", 10); // 7m

const SCENARIOS = {
  smoke: {
    executor: "shared-iterations",
    vus: parseInt(__ENV.SMOKE_VUS || "1", 10),
    iterations: parseInt(__ENV.SMOKE_ITERATIONS || "1", 10),
    maxDuration: __ENV.SMOKE_MAX_DURATION || "10m",
    tags: { phase: "smoke" },
  },
  load: {
    executor: "constant-vus",
    vus: LOAD_VUS,
    duration: LOAD_DURATION,
    gracefulStop: "30s",
    tags: { phase: "load" },
  },
  stress: {
    executor: "ramping-vus",
    startVUs: STRESS_START_VUS,
    stages: [
      { duration: STRESS_STEP_DURATION, target: STRESS_STEP_VUS },
      { duration: STRESS_STEP_DURATION, target: Math.min(STRESS_STEP_VUS * 2, STRESS_MAX_VUS) },
      { duration: STRESS_STEP_DURATION, target: STRESS_MAX_VUS },
      { duration: "1m", target: 0 },
    ],
    gracefulRampDown: "30s",
    tags: { phase: "stress" },
  },
  spike: {
    executor: "ramping-vus",
    startVUs: 0,
    stages: [
      { duration: SPIKE_RAMP_UP, target: SPIKE_VUS },
      { duration: SPIKE_HOLD, target: SPIKE_VUS },
      { duration: SPIKE_RAMP_DOWN, target: 0 },
    ],
    gracefulRampDown: "20s",
    tags: { phase: "spike" },
  },
  soak: {
    executor: "constant-vus",
    vus: SOAK_VUS,
    duration: SOAK_DURATION,
    gracefulStop: "1m",
    tags: { phase: "soak" },
  },
};

export const options = {
  scenarios: {
    [SCENARIOS[PHASE] ? PHASE : "load"]: SCENARIOS[SCENARIOS[PHASE] ? PHASE : "load"],
  },
  thresholds: {
    http_req_failed: ["rate<0.50"],
    smtp_route_failure: ["count<1000000"],
  },
};

const routeSuccess = new Counter("smtp_route_success");
const routeFailure = new Counter("smtp_route_failure");
const routeLatency = new Trend("smtp_route_latency_ms");
const refreshSuccess = new Counter("smtp_refresh_success");
const refreshFailure = new Counter("smtp_refresh_failure");

let vuSession = null;

export async function setup() {
  const state = {
    phase: PHASE,
    username: TEST_USERNAME,
    password: TEST_PASSWORD,
    workspaceId: WORKSPACE_ID_OVERRIDE,
    zoneId: ZONE_ID_OVERRIDE,
    fixtures: {},
  };

  const setupSession = await loginSession({
    username: TEST_USERNAME,
    password: TEST_PASSWORD,
    phase: "setup",
    deviceLabel: "setup-main",
  });

  state.workspaceId = await resolveOrCreateWorkspace(setupSession, state.workspaceId);
  setupSession.cookies.workspace_id = state.workspaceId;
  setupSession.workspaceId = state.workspaceId;

  state.zoneId = await resolveOrCreateZone(state.zoneId);
  setupSession.zoneId = state.zoneId;

  state.fixtures = await ensurePermanentFixtures(setupSession, state.zoneId);
  setupSession.fixtures = state.fixtures;

  await runFullCoverage(setupSession, "setup-smoke", true);

  return {
    username: state.username,
    password: state.password,
    workspaceId: state.workspaceId,
    zoneId: state.zoneId,
    fixtures: state.fixtures,
    cookies: { ...(setupSession.cookies || {}) },
  };
}

export default async function (state) {
  if (!vuSession) {
    vuSession = {
      phase: exec.scenario.name || state.phase || "load",
      workspaceId: state.workspaceId,
      zoneId: state.zoneId,
      fixtures: { ...(state.fixtures || {}) },
      cookies: { ...(state.cookies || {}) },
      lastRefreshAt: Date.now(),
    };
    vuSession.workspaceId = state.workspaceId;
    vuSession.zoneId = state.zoneId;
    vuSession.cookies.workspace_id = state.workspaceId;
  }

  const phase = exec.scenario.name || state.phase || "load";

  if (phase === "smoke") {
    await runFullCoverage(vuSession, phase, false);
    return;
  }

  await runTrafficMix(vuSession, phase);
  sleep(0.2 + Math.random() * 0.8);
}

export async function teardown(state) {
  const cleanupSession = {
    phase: "teardown",
    workspaceId: state.workspaceId,
    zoneId: state.zoneId,
    fixtures: { ...(state.fixtures || {}) },
    cookies: { ...(state.cookies || {}) },
  };
  cleanupSession.cookies.workspace_id = state.workspaceId;

  // Delete in dependency order.
  await deleteGatewayFixture(cleanupSession, state.fixtures.gateway, "teardown", false);
  await deleteTemplateFixture(cleanupSession, state.fixtures.template, "teardown", false);
  await deleteEndpointFixture(cleanupSession, state.fixtures.endpoint, "teardown", false);
  await deleteConsumerFixture(cleanupSession, state.fixtures.consumer, "teardown", false);
}

async function runTrafficMix(session, phase) {
  const roll = Math.random();

  if (roll < 0.10) {
    await getAggregation(session, phase);
    return;
  }
  if (roll < 0.20) {
    await getConsumerList(session, phase);
    await getTemplateList(session, phase);
    await getGatewayList(session, phase);
    await getEndpointList(session, phase);
    return;
  }
  if (roll < 0.32) {
    await getConsumerFixture(session, session.fixtures.consumer, phase);
    await getTemplateFixture(session, session.fixtures.template, phase);
    await getGatewayFixture(session, session.fixtures.gateway, phase);
    await getGatewayDetailFixture(session, session.fixtures.gateway, phase);
    await getEndpointFixture(session, session.fixtures.endpoint, phase);
    return;
  }
  if (roll < 0.40) {
    await getConsumerOptions(session, phase);
    await getRuntimeBatch(session, phase);
    return;
  }
  if (roll < 0.52) {
    await tryConnectConsumerFixture(session, session.zoneId, "traffic");
    await tryConnectEndpointFixture(session, "traffic");
    return;
  }
  if (roll < 0.66) {
    await updateConsumerFixture(session, session.fixtures.consumer, session.zoneId, "traffic", false);
    await updateTemplateFixture(session, session.fixtures.template, session.fixtures.consumer, "traffic", false);
    await updateEndpointFixture(session, session.fixtures.endpoint, "traffic", false);
    await updateGatewayFixture(
      session,
      session.fixtures.gateway,
      session.zoneId,
      session.fixtures.template,
      session.fixtures.endpoint,
      "traffic",
      false,
    );
    return;
  }
  if (roll < 0.82) {
    await startGatewayFixture(session, session.fixtures.gateway, phase, false);
    await drainGatewayFixture(session, session.fixtures.gateway, phase, false);
    await disableGatewayFixture(session, session.fixtures.gateway, phase, false);
    await startGatewayFixture(session, session.fixtures.gateway, phase, false);
    return;
  }

  await reconcileRuntime(session, phase);
}

async function runFullCoverage(session, phase, strict) {
  await getAggregation(session, phase);

  await getConsumerList(session, phase);
  await getConsumerOptions(session, phase);
  await getTemplateList(session, phase);
  await getGatewayList(session, phase);
  await getEndpointList(session, phase);

  await getConsumerFixture(session, session.fixtures.consumer, phase);
  await getTemplateFixture(session, session.fixtures.template, phase);
  await getGatewayFixture(session, session.fixtures.gateway, phase);
  await getGatewayDetailFixture(session, session.fixtures.gateway, phase);
  await getEndpointFixture(session, session.fixtures.endpoint, phase);

  await tryConnectConsumerFixture(session, session.zoneId, phase);
  await tryConnectEndpointFixture(session, phase);

  await updateConsumerFixture(session, session.fixtures.consumer, session.zoneId, phase, strict);
  await updateTemplateFixture(session, session.fixtures.template, session.fixtures.consumer, phase, strict);
  await updateEndpointFixture(session, session.fixtures.endpoint, phase, strict);
  await updateGatewayFixture(
    session,
    session.fixtures.gateway,
    session.zoneId,
    session.fixtures.template,
    session.fixtures.endpoint,
    phase,
    strict,
  );

  await startGatewayFixture(session, session.fixtures.gateway, phase, strict);
  await drainGatewayFixture(session, session.fixtures.gateway, phase, strict);
  await disableGatewayFixture(session, session.fixtures.gateway, phase, strict);
  await startGatewayFixture(session, session.fixtures.gateway, phase, strict);

  await getRuntimeBatch(session, phase);
  await reconcileRuntime(session, phase);

  // Full CRUD cycle on temporary resources to validate create + delete paths every run.
  const suffix = `${phase}-${Date.now()}-${randHex(4)}`;
  const tempConsumer = await createConsumerFixture(session, session.zoneId, `tmp-${suffix}`, strict);
  const tempEndpoint = await createEndpointFixture(session, `tmp-${suffix}`, strict);
  const tempTemplate = await createTemplateFixture(session, tempConsumer.id, `tmp-${suffix}`, strict);
  const tempGateway = await createGatewayFixture(session, session.zoneId, tempTemplate.id, tempEndpoint.id, `tmp-${suffix}`, strict);

  await getConsumerFixture(session, tempConsumer.id, phase);
  await getTemplateFixture(session, tempTemplate.id, phase);
  await getGatewayFixture(session, tempGateway.id, phase);
  await getGatewayDetailFixture(session, tempGateway.id, phase);
  await getEndpointFixture(session, tempEndpoint.id, phase);

  await deleteGatewayFixture(session, tempGateway.id, phase, strict);
  await deleteTemplateFixture(session, tempTemplate.id, phase, strict);
  await deleteEndpointFixture(session, tempEndpoint.id, phase, strict);
  await deleteConsumerFixture(session, tempConsumer.id, phase, strict);
}

async function getRuntimeBatch(session, phase) {
  await listRuntimeActivity(session, phase);
  await listRuntimeAttempts(session, phase);
  await listRuntimeHeartbeats(session, phase);
  await listGatewayAssignments(session, phase);
  await listConsumerAssignments(session, phase);
}

async function loginSession({ username, password, phase, deviceLabel }) {
  const device = await generateDeviceIdentity(deviceLabel);
  const session = {
    username,
    password,
    phase,
    workspaceId: "",
    zoneId: "",
    fixtures: {},
    cookies: {},
    deviceId: "",
    devicePrivateKey: device.privateKey,
    devicePublicKeyPem: device.publicKeyPem,
    deviceFingerprint: device.fingerprint,
    deviceAlgorithm: device.algorithm,
    lastRefreshAt: Date.now(),
  };

  const loginRes = await requestJson({
    method: "POST",
    route: "/api/v1/auth/login",
    phase,
    tags: smtpTags("/api/v1/auth/login", "POST", phase, "public"),
    body: {
      username,
      password,
      device_fingerprint: session.deviceFingerprint,
      device_public_key: session.devicePublicKeyPem,
      device_key_algorithm: session.deviceAlgorithm,
    },
    expected: [204],
    strict: true,
  });

  applySessionCookies(session, loginRes);
  session.deviceId = session.cookies.device_id || "";

  if (!session.cookies.access_token || !session.cookies.refresh_token || !session.deviceId) {
    fail(`login failed for ${username}: missing auth cookies`);
  }

  if (!session.cookies.refresh_token_hash && session.cookies.refresh_token) {
    session.cookies.refresh_token_hash = await hashRefreshToken(session.cookies.refresh_token);
  }

  return session;
}

async function refreshSession(session, phase, strict) {
  const jti = `smtp-refresh-${Date.now()}-${randHex(4)}`;
  const iat = Math.floor(Date.now() / 1000);
  const htm = "POST";
  const htu = `${BASE_URL}/api/v1/auth/refresh`;
  const tokenHash = session.cookies.refresh_token_hash || (await hashRefreshToken(session.cookies.refresh_token || ""));
  const deviceId = session.cookies.device_id || session.deviceId || "";

  if (!tokenHash || !deviceId) {
    refreshFailure.add(1, { phase });
    return null;
  }

  const signature = await signRefreshProof(session.devicePrivateKey, jti, iat, htm, htu, tokenHash, deviceId);
  const res = await requestJson({
    method: "POST",
    route: "/api/v1/auth/refresh",
    phase,
    tags: smtpTags("/api/v1/auth/refresh", "POST", phase, "session"),
    cookies: cookieHeaderFromSession(session),
    body: {
      jti,
      iat,
      htm,
      htu,
      token_hash: tokenHash,
      device_id: deviceId,
      signature,
    },
    expected: [204],
    strict,
  });

  if (res && res.status === 204) {
    applySessionCookies(session, res);
    session.lastRefreshAt = Date.now();
    refreshSuccess.add(1, { phase });
    return res;
  }

  refreshFailure.add(1, { phase });
  return res;
}

async function resolveOrCreateWorkspace(session, workspaceIdHint) {
  if (workspaceIdHint) {
    return workspaceIdHint;
  }

  const optionsRes = await requestJson({
    method: "GET",
    route: "/api/v1/workspaces/options",
    phase: "setup",
    tags: smtpTags("/api/v1/workspaces/options", "GET", "setup", "session"),
    cookies: cookieHeaderFromSession(session),
    expected: [200, 403],
    strict: false,
  });

  if (optionsRes.status === 200) {
    const data = responseData(optionsRes) || {};
    const items = data.items || [];
    if (items.length > 0) {
      const selected = items.find((item) => item && item.status === "active") || items[0];
      const selectedID = objectID(selected);
      if (selectedID) {
        return selectedID;
      }
    }
  }

  const suffix = `${Date.now()}-${randHex(3)}`;
  const createRes = await requestJson({
    method: "POST",
    route: "/api/v1/core/workspaces",
    phase: "setup",
    tags: smtpTags("/api/v1/core/workspaces", "POST", "setup", "bootstrap"),
    body: {
      name: `smtp-loadtest-${suffix}`,
      status: "active",
      tenant_id: "",
    },
    expected: [201],
    strict: true,
  });
  const data = responseData(createRes) || {};
  const createdID = objectID(data);
  if (!createdID) {
    fail("failed to resolve workspace id");
  }
  return createdID;
}

async function resolveOrCreateZone(zoneIdHint) {
  if (zoneIdHint) {
    return zoneIdHint;
  }

  const listRes = await requestJson({
    method: "GET",
    route: "/admin/core/zones",
    phase: "setup",
    tags: smtpTags("/admin/core/zones", "GET", "setup", "bootstrap"),
    expected: [200],
    strict: false,
  });
  const listData = responseData(listRes);
  if (Array.isArray(listData) && listData.length > 0) {
    const existingID = objectID(listData[0]);
    if (existingID) {
      return existingID;
    }
  }

  const suffix = `${Date.now()}-${randHex(3)}`;
  const createRes = await requestJson({
    method: "POST",
    route: "/admin/core/zones",
    phase: "setup",
    tags: smtpTags("/admin/core/zones", "POST", "setup", "bootstrap"),
    body: {
      slug: `smtp-loadtest-${suffix}`.toLowerCase(),
      name: `SMTP Loadtest ${suffix}`,
      description: "Zone for smtp load tests",
    },
    expected: [201],
    strict: true,
  });
  const data = responseData(createRes) || {};
  const createdID = objectID(data);
  if (!createdID) {
    fail("failed to resolve zone id");
  }
  return createdID;
}

async function ensurePermanentFixtures(session, zoneId) {
  const consumer = await createConsumerFixture(session, zoneId, "perm", true);
  const endpoint = await createEndpointFixture(session, "perm", true);
  const template = await createTemplateFixture(session, consumer.id, "perm", true);
  const gateway = await createGatewayFixture(session, zoneId, template.id, endpoint.id, "perm", true);

  await tryConnectConsumerFixture(session, zoneId, "perm");
  await tryConnectEndpointFixture(session, "perm");
  await reconcileRuntime(session, "setup");

  return {
    consumer: consumer.id,
    endpoint: endpoint.id,
    template: template.id,
    gateway: gateway.id,
  };
}

async function createConsumerFixture(session, zoneId, prefix, strict) {
  const body = consumerPayload(zoneId, prefix);
  const res = await requestJson({
    method: "POST",
    route: "/api/v1/smtp/consumers",
    phase: session.phase || "setup",
    tags: smtpTags("/api/v1/smtp/consumers", "POST", session.phase || "setup", "session"),
    cookies: cookieHeaderFromSession(session),
    body,
    expected: [201],
    strict,
  });
  const data = responseData(res) || {};
  if (!data.id && strict) {
    fail("create consumer did not return id");
  }
  return data;
}

async function updateConsumerFixture(session, consumerId, zoneId, prefix, strict) {
  const body = consumerPayload(zoneId, `${prefix}-updated`);
  body.name = `${body.name}-updated`;
  body.note = "updated by loadtest";
  return requestJson({
    method: "PUT",
    route: `/api/v1/smtp/consumers/${encodeURIComponent(consumerId)}`,
    phase: session.phase || "load",
    tags: smtpTags("/api/v1/smtp/consumers/:id", "PUT", session.phase || "load", "session"),
    cookies: cookieHeaderFromSession(session),
    body,
    expected: [200],
    strict,
  });
}

async function getConsumerList(session, phase) {
  return requestJson({
    method: "GET",
    route: "/api/v1/smtp/consumers",
    phase,
    tags: smtpTags("/api/v1/smtp/consumers", "GET", phase, "session"),
    cookies: cookieHeaderFromSession(session),
    expected: [200],
    strict: false,
  });
}

async function getConsumerFixture(session, consumerId, phase) {
  return requestJson({
    method: "GET",
    route: `/api/v1/smtp/consumers/${encodeURIComponent(consumerId)}`,
    phase,
    tags: smtpTags("/api/v1/smtp/consumers/:id", "GET", phase, "session"),
    cookies: cookieHeaderFromSession(session),
    expected: [200],
    strict: false,
  });
}

async function tryConnectConsumerFixture(session, zoneId, prefix) {
  const body = consumerPayload(zoneId, `${prefix}-probe`);
  body.name = `${body.name}-probe`;
  return requestJson({
    method: "POST",
    route: "/api/v1/smtp/consumers/try-connect",
    phase: session.phase || "load",
    tags: smtpTags("/api/v1/smtp/consumers/try-connect", "POST", session.phase || "load", "session"),
    cookies: cookieHeaderFromSession(session),
    body,
    expected: [200, 500, 503],
    strict: false,
  });
}

async function deleteConsumerFixture(session, consumerId, phase, strict) {
  if (!consumerId) {
    return null;
  }
  return requestJson({
    method: "DELETE",
    route: `/api/v1/smtp/consumers/${encodeURIComponent(consumerId)}`,
    phase,
    tags: smtpTags("/api/v1/smtp/consumers/:id", "DELETE", phase, "session"),
    cookies: cookieHeaderFromSession(session),
    expected: [200, 404],
    strict,
  });
}

async function getConsumerOptions(session, phase) {
  return requestJson({
    method: "GET",
    route: "/api/v1/smtp/consumers/options",
    phase,
    tags: smtpTags("/api/v1/smtp/consumers/options", "GET", phase, "session"),
    cookies: cookieHeaderFromSession(session),
    expected: [200],
    strict: false,
  });
}

async function createTemplateFixture(session, consumerId, prefix, strict) {
  const body = templatePayload(consumerId, prefix);
  const res = await requestJson({
    method: "POST",
    route: "/api/v1/smtp/templates",
    phase: session.phase || "setup",
    tags: smtpTags("/api/v1/smtp/templates", "POST", session.phase || "setup", "session"),
    cookies: cookieHeaderFromSession(session),
    body,
    expected: [201],
    strict,
  });
  const data = responseData(res) || {};
  if (!data.id && strict) {
    fail("create template did not return id");
  }
  return data;
}

async function updateTemplateFixture(session, templateId, consumerId, prefix, strict) {
  const body = templatePayload(consumerId, `${prefix}-updated`);
  body.name = `${body.name}-updated`;
  body.subject = `[SMTP Loadtest] ${Date.now()}`;
  return requestJson({
    method: "PUT",
    route: `/api/v1/smtp/templates/${encodeURIComponent(templateId)}`,
    phase: session.phase || "load",
    tags: smtpTags("/api/v1/smtp/templates/:id", "PUT", session.phase || "load", "session"),
    cookies: cookieHeaderFromSession(session),
    body,
    expected: [200],
    strict,
  });
}

async function getTemplateList(session, phase) {
  return requestJson({
    method: "GET",
    route: "/api/v1/smtp/templates",
    phase,
    tags: smtpTags("/api/v1/smtp/templates", "GET", phase, "session"),
    cookies: cookieHeaderFromSession(session),
    expected: [200],
    strict: false,
  });
}

async function getTemplateFixture(session, templateId, phase) {
  return requestJson({
    method: "GET",
    route: `/api/v1/smtp/templates/${encodeURIComponent(templateId)}`,
    phase,
    tags: smtpTags("/api/v1/smtp/templates/:id", "GET", phase, "session"),
    cookies: cookieHeaderFromSession(session),
    expected: [200],
    strict: false,
  });
}

async function deleteTemplateFixture(session, templateId, phase, strict) {
  if (!templateId) {
    return null;
  }
  return requestJson({
    method: "DELETE",
    route: `/api/v1/smtp/templates/${encodeURIComponent(templateId)}`,
    phase,
    tags: smtpTags("/api/v1/smtp/templates/:id", "DELETE", phase, "session"),
    cookies: cookieHeaderFromSession(session),
    expected: [200, 404],
    strict,
  });
}

async function createEndpointFixture(session, prefix, strict) {
  const body = endpointPayload(prefix);
  const res = await requestJson({
    method: "POST",
    route: "/api/v1/smtp/endpoints",
    phase: session.phase || "setup",
    tags: smtpTags("/api/v1/smtp/endpoints", "POST", session.phase || "setup", "session"),
    cookies: cookieHeaderFromSession(session),
    body,
    expected: [201],
    strict,
  });
  const data = responseData(res) || {};
  if (!data.id && strict) {
    fail("create endpoint did not return id");
  }
  return data;
}

async function updateEndpointFixture(session, endpointId, prefix, strict) {
  const body = endpointPayload(`${prefix}-updated`);
  body.name = `${body.name}-updated`;
  body.priority = 20;
  return requestJson({
    method: "PUT",
    route: `/api/v1/smtp/endpoints/${encodeURIComponent(endpointId)}`,
    phase: session.phase || "load",
    tags: smtpTags("/api/v1/smtp/endpoints/:id", "PUT", session.phase || "load", "session"),
    cookies: cookieHeaderFromSession(session),
    body,
    expected: [200],
    strict,
  });
}

async function getEndpointList(session, phase) {
  return requestJson({
    method: "GET",
    route: "/api/v1/smtp/endpoints",
    phase,
    tags: smtpTags("/api/v1/smtp/endpoints", "GET", phase, "session"),
    cookies: cookieHeaderFromSession(session),
    expected: [200],
    strict: false,
  });
}

async function getEndpointFixture(session, endpointId, phase) {
  return requestJson({
    method: "GET",
    route: `/api/v1/smtp/endpoints/${encodeURIComponent(endpointId)}`,
    phase,
    tags: smtpTags("/api/v1/smtp/endpoints/:id", "GET", phase, "session"),
    cookies: cookieHeaderFromSession(session),
    expected: [200],
    strict: false,
  });
}

async function tryConnectEndpointFixture(session, prefix) {
  const body = endpointPayload(`${prefix}-probe`);
  body.name = `${body.name}-probe`;
  return requestJson({
    method: "POST",
    route: "/api/v1/smtp/endpoints/try-connect",
    phase: session.phase || "load",
    tags: smtpTags("/api/v1/smtp/endpoints/try-connect", "POST", session.phase || "load", "session"),
    cookies: cookieHeaderFromSession(session),
    body,
    expected: [200, 500, 503],
    strict: false,
  });
}

async function deleteEndpointFixture(session, endpointId, phase, strict) {
  if (!endpointId) {
    return null;
  }
  return requestJson({
    method: "DELETE",
    route: `/api/v1/smtp/endpoints/${encodeURIComponent(endpointId)}`,
    phase,
    tags: smtpTags("/api/v1/smtp/endpoints/:id", "DELETE", phase, "session"),
    cookies: cookieHeaderFromSession(session),
    expected: [200, 404],
    strict,
  });
}

async function createGatewayFixture(session, zoneId, templateId, endpointId, prefix, strict) {
  const body = gatewayPayload(zoneId, templateId, endpointId, prefix);
  const res = await requestJson({
    method: "POST",
    route: "/api/v1/smtp/gateways",
    phase: session.phase || "setup",
    tags: smtpTags("/api/v1/smtp/gateways", "POST", session.phase || "setup", "session"),
    cookies: cookieHeaderFromSession(session),
    body,
    expected: [201],
    strict,
  });
  const data = responseData(res) || {};
  if (!data.id && strict) {
    fail("create gateway did not return id");
  }
  return data;
}

async function updateGatewayFixture(session, gatewayId, zoneId, templateId, endpointId, prefix, strict) {
  const body = gatewayPayload(zoneId, templateId, endpointId, `${prefix}-updated`);
  body.name = `${body.name}-updated`;
  body.priority = 20;
  await requestJson({
    method: "PUT",
    route: `/api/v1/smtp/gateways/${encodeURIComponent(gatewayId)}`,
    phase: session.phase || "load",
    tags: smtpTags("/api/v1/smtp/gateways/:id", "PUT", session.phase || "load", "session"),
    cookies: cookieHeaderFromSession(session),
    body,
    expected: [200],
    strict,
  });
  await requestJson({
    method: "PUT",
    route: `/api/v1/smtp/gateways/${encodeURIComponent(gatewayId)}/templates`,
    phase: session.phase || "load",
    tags: smtpTags("/api/v1/smtp/gateways/:id/templates", "PUT", session.phase || "load", "session"),
    cookies: cookieHeaderFromSession(session),
    body: { template_ids: [templateId] },
    expected: [200],
    strict,
  });
  await requestJson({
    method: "PUT",
    route: `/api/v1/smtp/gateways/${encodeURIComponent(gatewayId)}/endpoints`,
    phase: session.phase || "load",
    tags: smtpTags("/api/v1/smtp/gateways/:id/endpoints", "PUT", session.phase || "load", "session"),
    cookies: cookieHeaderFromSession(session),
    body: { endpoint_ids: [endpointId] },
    expected: [200],
    strict,
  });
}

async function getGatewayList(session, phase) {
  return requestJson({
    method: "GET",
    route: "/api/v1/smtp/gateways",
    phase,
    tags: smtpTags("/api/v1/smtp/gateways", "GET", phase, "session"),
    cookies: cookieHeaderFromSession(session),
    expected: [200],
    strict: false,
  });
}

async function getGatewayFixture(session, gatewayId, phase) {
  return requestJson({
    method: "GET",
    route: `/api/v1/smtp/gateways/${encodeURIComponent(gatewayId)}`,
    phase,
    tags: smtpTags("/api/v1/smtp/gateways/:id", "GET", phase, "session"),
    cookies: cookieHeaderFromSession(session),
    expected: [200],
    strict: false,
  });
}

async function getGatewayDetailFixture(session, gatewayId, phase) {
  return requestJson({
    method: "GET",
    route: `/api/v1/smtp/gateways/${encodeURIComponent(gatewayId)}/detail`,
    phase,
    tags: smtpTags("/api/v1/smtp/gateways/:id/detail", "GET", phase, "session"),
    cookies: cookieHeaderFromSession(session),
    expected: [200],
    strict: false,
  });
}

async function startGatewayFixture(session, gatewayId, phase, strict) {
  return requestJson({
    method: "POST",
    route: `/api/v1/smtp/gateways/${encodeURIComponent(gatewayId)}/start`,
    phase,
    tags: smtpTags("/api/v1/smtp/gateways/:id/start", "POST", phase, "session"),
    cookies: cookieHeaderFromSession(session),
    expected: [200, 409],
    strict,
  });
}

async function drainGatewayFixture(session, gatewayId, phase, strict) {
  return requestJson({
    method: "POST",
    route: `/api/v1/smtp/gateways/${encodeURIComponent(gatewayId)}/drain`,
    phase,
    tags: smtpTags("/api/v1/smtp/gateways/:id/drain", "POST", phase, "session"),
    cookies: cookieHeaderFromSession(session),
    expected: [200, 409],
    strict,
  });
}

async function disableGatewayFixture(session, gatewayId, phase, strict) {
  return requestJson({
    method: "POST",
    route: `/api/v1/smtp/gateways/${encodeURIComponent(gatewayId)}/disable`,
    phase,
    tags: smtpTags("/api/v1/smtp/gateways/:id/disable", "POST", phase, "session"),
    cookies: cookieHeaderFromSession(session),
    expected: [200, 409],
    strict,
  });
}

async function deleteGatewayFixture(session, gatewayId, phase, strict) {
  if (!gatewayId) {
    return null;
  }
  return requestJson({
    method: "DELETE",
    route: `/api/v1/smtp/gateways/${encodeURIComponent(gatewayId)}`,
    phase,
    tags: smtpTags("/api/v1/smtp/gateways/:id", "DELETE", phase, "session"),
    cookies: cookieHeaderFromSession(session),
    expected: [200, 404],
    strict,
  });
}

async function getAggregation(session, phase) {
  return requestJson({
    method: "GET",
    route: "/api/v1/smtp/aggregation",
    phase,
    tags: smtpTags("/api/v1/smtp/aggregation", "GET", phase, "session"),
    cookies: cookieHeaderFromSession(session),
    expected: [200],
    strict: false,
  });
}

async function listRuntimeActivity(session, phase) {
  return requestJson({
    method: "GET",
    route: "/api/v1/smtp/runtime/activity-logs",
    phase,
    tags: smtpTags("/api/v1/smtp/runtime/activity-logs", "GET", phase, "session"),
    cookies: cookieHeaderFromSession(session),
    expected: [200],
    strict: false,
  });
}

async function listRuntimeAttempts(session, phase) {
  return requestJson({
    method: "GET",
    route: "/api/v1/smtp/runtime/delivery-attempts",
    phase,
    tags: smtpTags("/api/v1/smtp/runtime/delivery-attempts", "GET", phase, "session"),
    cookies: cookieHeaderFromSession(session),
    expected: [200],
    strict: false,
  });
}

async function listRuntimeHeartbeats(session, phase) {
  return requestJson({
    method: "GET",
    route: "/api/v1/smtp/runtime/heartbeats",
    phase,
    tags: smtpTags("/api/v1/smtp/runtime/heartbeats", "GET", phase, "session"),
    cookies: cookieHeaderFromSession(session),
    expected: [200],
    strict: false,
  });
}

async function listGatewayAssignments(session, phase) {
  return requestJson({
    method: "GET",
    route: "/api/v1/smtp/runtime/gateway-assignments",
    phase,
    tags: smtpTags("/api/v1/smtp/runtime/gateway-assignments", "GET", phase, "session"),
    cookies: cookieHeaderFromSession(session),
    expected: [200],
    strict: false,
  });
}

async function listConsumerAssignments(session, phase) {
  return requestJson({
    method: "GET",
    route: "/api/v1/smtp/runtime/consumer-assignments",
    phase,
    tags: smtpTags("/api/v1/smtp/runtime/consumer-assignments", "GET", phase, "session"),
    cookies: cookieHeaderFromSession(session),
    expected: [200],
    strict: false,
  });
}

async function reconcileRuntime(session, phase) {
  return requestJson({
    method: "POST",
    route: "/api/v1/smtp/runtime/reconcile",
    phase,
    tags: smtpTags("/api/v1/smtp/runtime/reconcile", "POST", phase, "session"),
    cookies: cookieHeaderFromSession(session),
    expected: [200],
    strict: false,
  });
}

function consumerPayload(zoneId, prefix) {
  return {
    zone_id: zoneId,
    name: `smtp-${prefix}-consumer`,
    transport_type: "redis_stream",
    source: "smtp.outbound",
    consumer_group: "smtp-loadtest-group",
    worker_concurrency: 2,
    ack_timeout_seconds: 5,
    batch_size: 50,
    status: "active",
    note: "smtp module load test consumer",
    connection_config: {
      addr: "127.0.0.1:6379",
      db: 0,
      tls_mode: "none",
    },
    desired_shard_count: 1,
    secret_config: {},
    secret_ref: "",
    secret_provider: "",
  };
}

function templatePayload(consumerId, prefix) {
  return {
    name: `smtp-${prefix}-template`,
    category: "transactional",
    traffic_class: "transactional",
    subject: `[SMTP Loadtest] ${prefix}`,
    from_email: "loadtest@example.test",
    to_email: "recipient@example.test",
    status: "live",
    variables: ["name"],
    consumer_id: consumerId,
    retry_max_attempts: 3,
    retry_backoff_seconds: 5,
    text_body: "Hello {{name}}",
    html_body: "<p>Hello {{name}}</p>",
  };
}

function gatewayPayload(zoneId, templateId, endpointId, prefix) {
  return {
    zone_id: zoneId,
    name: `smtp-${prefix}-gateway`,
    traffic_class: "transactional",
    status: "active",
    routing_mode: "round_robin",
    priority: 10,
    fallback_gateway_id: "",
    desired_shard_count: 1,
    template_ids: [templateId],
    endpoint_ids: [endpointId],
  };
}

function endpointPayload(prefix) {
  return {
    name: `smtp-${prefix}-endpoint`,
    provider_kind: "smtp",
    host: SMTP_MOCK_HOST,
    port: SMTP_MOCK_PORT,
    username: "",
    priority: 10,
    weight: 10,
    max_connections: 20,
    max_parallel_sends: 10,
    max_messages_per_second: 25,
    burst: 10,
    warmup_state: "stable",
    status: "active",
    tls_mode: "none",
    password: "",
    ca_cert_pem: null,
    client_cert_pem: null,
    client_key_pem: null,
    secret_ref: null,
    secret_provider: null,
  };
}

async function requestJson({ method, route, body, cookies, tags, expected = [200], strict = false, phase = "load" }) {
  const url = `${BASE_URL}${route}`;
  const headers = { Accept: "application/json" };
  if (body !== undefined && body !== null) {
    headers["Content-Type"] = "application/json";
  }
  if (cookies) {
    headers.Cookie = cookies;
  }

  const payload = body === undefined || body === null ? null : JSON.stringify(body);
  const res = http.request(method, url, payload, { headers, tags });
  routeLatency.add(res.timings.duration, tags || { phase });

  if (expected.includes(res.status)) {
    routeSuccess.add(1, tags || { phase });
  } else {
    routeFailure.add(1, tags || { phase });
    if (strict) {
      fail(`${method} ${route} expected ${expected.join(",")} but got ${res.status}: ${res.body}`);
    }
  }

  return res;
}

function responseData(res) {
  if (!res || !res.body || res.status === 204) {
    return null;
  }
  try {
    const parsed = res.json();
    if (parsed && Object.prototype.hasOwnProperty.call(parsed, "data")) {
      return parsed.data;
    }
    return parsed;
  } catch (_) {
    return null;
  }
}

function applySessionCookies(session, res) {
  if (!session || !res || !res.cookies) {
    return;
  }
  const names = ["access_token", "refresh_token", "device_id", "refresh_token_hash"];
  for (const name of names) {
    const value = firstCookieValue(res, name);
    if (value !== "") {
      session.cookies[name] = value;
    }
  }
}

function firstCookieValue(res, name) {
  if (!res || !res.cookies || !res.cookies[name] || !res.cookies[name].length) {
    return "";
  }
  const cookie = res.cookies[name][0];
  return cookie && cookie.value ? cookie.value : "";
}

function cookieHeaderFromSession(session) {
  const keys = ["access_token", "refresh_token", "device_id", "refresh_token_hash", "workspace_id"];
  const parts = [];
  for (const key of keys) {
    const value = session && session.cookies ? session.cookies[key] : "";
    if (value) {
      parts.push(`${key}=${value}`);
    }
  }
  return parts.join("; ");
}

function smtpTags(route, method, phase, authClass) {
  return {
    module: route.startsWith("/api/v1/smtp") ? "smtp" : "bootstrap",
    route,
    method,
    phase,
    auth_class: authClass,
  };
}

function objectID(item) {
  if (!item || typeof item !== "object") {
    return "";
  }
  return trim(item.id || item.ID || "");
}

function trim(value) {
  return String(value || "").trim();
}

function trimSlash(value) {
  return trim(value).replace(/\/+$/, "");
}

function randHex(bytes) {
  const out = new Uint8Array(bytes);
  crypto.getRandomValues(out);
  let s = "";
  for (let i = 0; i < out.length; i++) {
    s += out[i].toString(16).padStart(2, "0");
  }
  return s;
}

function bytesToBase64Url(bytes) {
  return b64encode(bytes, "rawurl");
}

function bytesToBase64Std(bytes) {
  return b64encode(bytes, "std");
}

function wrapPem(body) {
  const lines = [];
  for (let i = 0; i < body.length; i += 64) {
    lines.push(body.slice(i, i + 64));
  }
  return lines.join("\n");
}

function pemFromSpki(spkiBytes) {
  return [
    "-----BEGIN PUBLIC KEY-----",
    wrapPem(bytesToBase64Std(spkiBytes)),
    "-----END PUBLIC KEY-----",
    "",
  ].join("\n");
}

async function generateDeviceIdentity(label) {
  const keyPair = await crypto.subtle.generateKey(
    { name: "ECDSA", namedCurve: "P-256" },
    true,
    ["sign", "verify"],
  );
  const spki = new Uint8Array(await crypto.subtle.exportKey("spki", keyPair.publicKey));
  return {
    privateKey: keyPair.privateKey,
    publicKeyPem: pemFromSpki(spki),
    fingerprint: `smtp-${label}-${Date.now()}-${randHex(6)}`,
    algorithm: "ES256",
  };
}

async function signRefreshProof(privateKey, jti, issuedAt, htm, htu, tokenHash, deviceId) {
  const payload = [jti, String(issuedAt), htm.toUpperCase(), htu, tokenHash, deviceId].join("\n");
  const encoded = new TextEncoder().encode(payload);
  const signature = new Uint8Array(
    await crypto.subtle.sign(
      { name: "ECDSA", hash: "SHA-256" },
      privateKey,
      encoded,
    ),
  );
  return bytesToBase64Url(derToRawEs256(signature));
}

function derToRawEs256(signatureBytes) {
  if (signatureBytes.length === 64) {
    return signatureBytes;
  }
  if (signatureBytes.length < 8 || signatureBytes[0] !== 0x30) {
    fail("unexpected ECDSA signature encoding");
  }

  let offset = 1;
  let seqLen = signatureBytes[offset++];
  if (seqLen & 0x80) {
    const lengthBytes = seqLen & 0x7f;
    seqLen = 0;
    for (let i = 0; i < lengthBytes; i++) {
      seqLen = (seqLen << 8) | signatureBytes[offset++];
    }
  }

  if (signatureBytes[offset++] !== 0x02) {
    fail("invalid ECDSA signature integer tag for r");
  }
  const rLen = readAsn1Length(signatureBytes, { value: offset }, (nextOffset) => {
    offset = nextOffset;
  });
  const r = signatureBytes.slice(offset, offset + rLen);
  offset += rLen;

  if (signatureBytes[offset++] !== 0x02) {
    fail("invalid ECDSA signature integer tag for s");
  }
  const sLen = readAsn1Length(signatureBytes, { value: offset }, (nextOffset) => {
    offset = nextOffset;
  });
  const s = signatureBytes.slice(offset, offset + sLen);

  return concatPad32(r, s);
}

function readAsn1Length(bytes, offsetRef, setOffset) {
  let offset = offsetRef.value;
  let len = bytes[offset++];
  if (!(len & 0x80)) {
    setOffset(offset);
    return len;
  }
  const count = len & 0x7f;
  len = 0;
  for (let i = 0; i < count; i++) {
    len = (len << 8) | bytes[offset++];
  }
  setOffset(offset);
  return len;
}

function concatPad32(r, s) {
  const out = new Uint8Array(64);
  out.set(leftPad32(r), 0);
  out.set(leftPad32(s), 32);
  return out;
}

function leftPad32(value) {
  const out = new Uint8Array(32);
  const start = Math.max(0, value.length - 32);
  const copy = value.slice(start);
  out.set(copy, 32 - copy.length);
  return out;
}

async function hashRefreshToken(refreshToken) {
  const normalized = trim(refreshToken);
  if (!normalized) {
    return "";
  }
  const digest = new Uint8Array(await crypto.subtle.digest("SHA-256", new TextEncoder().encode(normalized)));
  return bytesToBase64Url(digest);
}
