import exec from "k6/execution";
import http from "k6/http";
import { Counter, Trend } from "k6/metrics";
import { b64encode } from "k6/encoding";
import { sleep } from "k6";

const BASE_URL = trimSlash(__ENV.BASE_URL || "http://127.0.0.1:8080");
const TEST_USERNAME = trim(__ENV.TEST_USERNAME || "iam.loadtest");
const TEST_PASSWORD = __ENV.TEST_PASSWORD || "Loadtest123!";
const PHASE = trim(__ENV.PHASE || "load").toLowerCase();

const LOAD_VUS = parseInt(__ENV.LOAD_VUS || "100", 10);
const LOAD_DURATION = __ENV.LOAD_DURATION || "5m";

const STRESS_START_VUS = parseInt(__ENV.STRESS_START_VUS || "20", 10);
const STRESS_MAX_VUS = parseInt(__ENV.STRESS_MAX_VUS || "300", 10);
const STRESS_STEP_DURATION = __ENV.STRESS_STEP_DURATION || "2m";

const SPIKE_VUS = parseInt(__ENV.SPIKE_VUS || "500", 10);
const SPIKE_RAMP_UP = __ENV.SPIKE_RAMP_UP || "30s";
const SPIKE_HOLD = __ENV.SPIKE_HOLD || "90s";
const SPIKE_RAMP_DOWN = __ENV.SPIKE_RAMP_DOWN || "30s";

const SOAK_VUS = parseInt(__ENV.SOAK_VUS || "50", 10);
const SOAK_DURATION = __ENV.SOAK_DURATION || "30m";

const AUTH_LANE_VUS = parseInt(__ENV.AUTH_LANE_VUS || "1", 10);
const AUTH_ITER_SLEEP_SEC = parseFloat(__ENV.AUTH_ITER_SLEEP_SEC || "20");
const BASELINE_SLEEP_MIN = parseFloat(__ENV.BASELINE_SLEEP_MIN || "0.1");
const BASELINE_SLEEP_MAX = parseFloat(__ENV.BASELINE_SLEEP_MAX || "0.4");

const routeSuccess = new Counter("iam_route_success");
const routeFailure = new Counter("iam_route_failure");
const routeLatency = new Trend("iam_route_latency_ms");
const refreshSuccess = new Counter("iam_refresh_success");
const refreshFailure = new Counter("iam_refresh_failure");

const scenariosByPhase = {
  load: {
    load_baseline: {
      executor: "constant-vus",
      vus: LOAD_VUS,
      duration: LOAD_DURATION,
      gracefulStop: "30s",
      tags: { phase: "load", lane: "baseline" },
    },
    load_auth_limited: {
      executor: "constant-vus",
      vus: AUTH_LANE_VUS,
      duration: LOAD_DURATION,
      gracefulStop: "30s",
      tags: { phase: "load", lane: "auth_limited" },
    },
  },
  stress: {
    stress_baseline: {
      executor: "ramping-vus",
      startVUs: STRESS_START_VUS,
      stages: [
        { duration: STRESS_STEP_DURATION, target: Math.floor(STRESS_MAX_VUS / 3) },
        { duration: STRESS_STEP_DURATION, target: Math.floor((STRESS_MAX_VUS * 2) / 3) },
        { duration: STRESS_STEP_DURATION, target: STRESS_MAX_VUS },
        { duration: "1m", target: 0 },
      ],
      gracefulRampDown: "30s",
      tags: { phase: "stress", lane: "baseline" },
    },
    stress_auth_limited: {
      executor: "constant-vus",
      vus: AUTH_LANE_VUS,
      duration: "7m",
      gracefulStop: "30s",
      tags: { phase: "stress", lane: "auth_limited" },
    },
  },
  spike: {
    spike_baseline: {
      executor: "ramping-vus",
      startVUs: 0,
      stages: [
        { duration: SPIKE_RAMP_UP, target: SPIKE_VUS },
        { duration: SPIKE_HOLD, target: SPIKE_VUS },
        { duration: SPIKE_RAMP_DOWN, target: 0 },
      ],
      gracefulRampDown: "20s",
      tags: { phase: "spike", lane: "baseline" },
    },
    spike_auth_limited: {
      executor: "constant-vus",
      vus: AUTH_LANE_VUS,
      duration: "3m",
      gracefulStop: "20s",
      tags: { phase: "spike", lane: "auth_limited" },
    },
  },
  soak: {
    soak_baseline: {
      executor: "constant-vus",
      vus: SOAK_VUS,
      duration: SOAK_DURATION,
      gracefulStop: "1m",
      tags: { phase: "soak", lane: "baseline" },
    },
    soak_auth_limited: {
      executor: "constant-vus",
      vus: AUTH_LANE_VUS,
      duration: SOAK_DURATION,
      gracefulStop: "1m",
      tags: { phase: "soak", lane: "auth_limited" },
    },
  },
};

export const options = {
  scenarios: scenariosByPhase[PHASE] || scenariosByPhase.load,
  thresholds: {
    http_req_failed: ["rate<0.02"],
    iam_route_failure: ["count<1000000"],
  },
};

let vuState = null;

export async function setup() {
  const baseline = await performLogin(TEST_USERNAME, TEST_PASSWORD, "setup");
  if (!baseline.ok) {
    throw new Error(`setup login failed: status=${baseline.status}`);
  }

  const whoami = request("GET", "/api/v1/whoami", {
    cookies: baseline.cookies,
    expected: [200],
    tags: { route: "/api/v1/whoami", lane: "setup", auth_class: "access" },
  });
  if (!isExpected(whoami, [200])) {
    throw new Error(`setup whoami failed: status=${whoami.status}`);
  }

  return { baselineCookies: baseline.cookies };
}

export default async function (data) {
  if (!vuState) {
    vuState = {
      baselineCookies: { ...(data.baselineCookies || {}) },
    };
  }

  const scenario = exec.scenario.name || "unknown";
  if (scenario.includes("auth_limited")) {
    await runAuthLimitedLane();
    sleep(AUTH_ITER_SLEEP_SEC);
    return;
  }

  runBaselineLane(vuState.baselineCookies);
  sleep(BASELINE_SLEEP_MIN + Math.random() * (BASELINE_SLEEP_MAX - BASELINE_SLEEP_MIN));
}

function runBaselineLane(cookies) {
  request("GET", "/api/v1/whoami", {
    cookies,
    expected: [200, 401],
    tags: { route: "/api/v1/whoami", lane: "baseline", auth_class: "access" },
  });
  request("GET", "/api/v1/me/devices", {
    cookies,
    expected: [200, 401],
    tags: { route: "/api/v1/me/devices", lane: "baseline", auth_class: "access_device" },
  });
}

async function runAuthLimitedLane() {
  const login = await performLogin(TEST_USERNAME, TEST_PASSWORD, `auth-${exec.vu.idInTest}-${Date.now()}`);
  if (!login.ok) {
    return;
  }

  request("GET", "/api/v1/whoami", {
    cookies: login.cookies,
    expected: [200],
    tags: { route: "/api/v1/whoami", lane: "auth_limited", auth_class: "access" },
  });
  request("GET", "/api/v1/me/devices", {
    cookies: login.cookies,
    expected: [200],
    tags: { route: "/api/v1/me/devices", lane: "auth_limited", auth_class: "access_device" },
  });

  const refreshProof = await buildRefreshProof(login.privateKey, login.cookies);
  if (refreshProof == null) {
    return;
  }

  const refresh = request("POST", "/api/v1/auth/refresh", {
    cookies: login.cookies,
    expected: [204, 401, 429],
    body: JSON.stringify(refreshProof),
    tags: { route: "/api/v1/auth/refresh", lane: "auth_limited", auth_class: "refresh_proof" },
  });
  if (refresh.status === 204) {
    refreshSuccess.add(1);
    mergeCookies(login.cookies, parseSetCookies(refresh));
  } else {
    refreshFailure.add(1);
  }

  request("GET", "/api/v1/whoami", {
    cookies: login.cookies,
    expected: [200, 401],
    tags: { route: "/api/v1/whoami", lane: "auth_limited", auth_class: "access" },
  });

  request("POST", "/api/v1/auth/logout", {
    cookies: login.cookies,
    expected: [200, 401],
    tags: { route: "/api/v1/auth/logout", lane: "auth_limited", auth_class: "access_device" },
  });
}

async function performLogin(username, password, label) {
  const identity = await generateDeviceIdentity(label);
  const loginPayload = {
    username,
    password,
    device_fingerprint: identity.fingerprint,
    device_public_key: identity.publicKeyPem,
    device_key_algorithm: "ES256",
  };

  const loginRes = request("POST", "/api/v1/auth/login", {
    expected: [204, 202, 401, 403, 429],
    body: JSON.stringify(loginPayload),
    tags: { route: "/api/v1/auth/login", lane: "auth_limited", auth_class: "public" },
  });

  if (loginRes.status !== 204) {
    return {
      ok: false,
      status: loginRes.status,
      cookies: {},
      privateKey: null,
    };
  }

  const cookies = parseSetCookies(loginRes);
  return {
    ok: true,
    status: loginRes.status,
    cookies,
    privateKey: identity.privateKey,
  };
}

async function buildRefreshProof(privateKey, cookies) {
  if (privateKey == null) {
    return null;
  }
  const deviceId = cookies.device_id || "";
  const tokenHash = cookies.refresh_token_hash || "";
  if (deviceId === "" || tokenHash === "") {
    return null;
  }

  const jti = randomUUID();
  const iat = Math.floor(Date.now() / 1000);
  const htm = "POST";
  const htu = `${BASE_URL}/api/v1/auth/refresh`;
  const payload = [jti, String(iat), htm, htu, tokenHash, deviceId].join("\n");
  const signature = await signPayload(privateKey, payload);

  return {
    jti,
    iat,
    htm,
    htu,
    token_hash: tokenHash,
    device_id: deviceId,
    signature,
  };
}

function request(method, path, cfg) {
  const url = `${BASE_URL}${path}`;
  const tags = {
    module: "iam",
    route: cfg.tags?.route || path,
    lane: cfg.tags?.lane || "baseline",
    auth_class: cfg.tags?.auth_class || "public",
    method,
    phase: PHASE,
  };
  const headers = {
    Accept: "application/json",
    ...(cfg.body ? { "Content-Type": "application/json" } : {}),
    ...(cfg.cookies ? { Cookie: buildCookieHeader(cfg.cookies) } : {}),
  };

  const params = { headers, tags };
  const res = http.request(method, url, cfg.body || null, params);
  routeLatency.add(res.timings.duration, tags);

  const expected = cfg.expected || [200];
  if (isExpected(res, expected)) {
    routeSuccess.add(1, tags);
  } else {
    routeFailure.add(1, {
      ...tags,
      status: String(res.status),
    });
  }
  return res;
}

function isExpected(res, expected) {
  return Array.isArray(expected) && expected.includes(res.status);
}

function parseSetCookies(res) {
  const out = {};
  const raw = res.headers["Set-Cookie"];
  if (!raw) {
    return out;
  }

  const values = Array.isArray(raw) ? raw : [raw];
  for (const value of values) {
    const first = String(value).split(";")[0];
    const idx = first.indexOf("=");
    if (idx <= 0) {
      continue;
    }
    const name = first.slice(0, idx).trim();
    const cookieValue = first.slice(idx + 1).trim();
    out[name] = cookieValue;
  }
  return out;
}

function mergeCookies(target, source) {
  if (!target || !source) {
    return;
  }
  for (const [name, value] of Object.entries(source)) {
    target[name] = value;
  }
}

function buildCookieHeader(cookies) {
  const pairs = [];
  for (const [name, value] of Object.entries(cookies || {})) {
    if (!name || value === undefined || value === null || String(value) === "") {
      continue;
    }
    pairs.push(`${name}=${value}`);
  }
  return pairs.join("; ");
}

async function generateDeviceIdentity(label) {
  const keyPair = await crypto.subtle.generateKey(
    {
      name: "ECDSA",
      namedCurve: "P-256",
    },
    true,
    ["sign", "verify"],
  );

  const spki = await crypto.subtle.exportKey("spki", keyPair.publicKey);
  const publicKeyPem = toPEM(spki);
  const fingerprint = `${label}-${randomUUID()}`;

  return {
    fingerprint,
    publicKeyPem,
    privateKey: keyPair.privateKey,
  };
}

async function signPayload(privateKey, payload) {
  const signature = await crypto.subtle.sign(
    {
      name: "ECDSA",
      hash: "SHA-256",
    },
    privateKey,
    new TextEncoder().encode(payload),
  );

  const raw = ecdsaSignatureToRaw(new Uint8Array(signature));
  return base64UrlEncode(raw);
}

function ecdsaSignatureToRaw(signature) {
  if (signature.length === 64) {
    return signature;
  }

  let offset = 0;
  if (signature[offset++] !== 0x30) {
    throw new Error("unexpected ecdsa signature format");
  }
  const seqLength = readDerLength(signature, offset);
  offset += seqLength.bytesRead;

  if (signature[offset++] !== 0x02) {
    throw new Error("unexpected ecdsa signature format");
  }
  const rLength = readDerLength(signature, offset);
  offset += rLength.bytesRead;
  const r = signature.slice(offset, offset + rLength.length);
  offset += rLength.length;

  if (signature[offset++] !== 0x02) {
    throw new Error("unexpected ecdsa signature format");
  }
  const sLength = readDerLength(signature, offset);
  offset += sLength.bytesRead;
  const s = signature.slice(offset, offset + sLength.length);
  offset += sLength.length;

  if (offset !== signature.length) {
    throw new Error("unexpected ecdsa signature format");
  }

  return concatFixedLengthIntegers(r, s, 32);
}

function readDerLength(buffer, offset) {
  const first = buffer[offset];
  if (first === undefined) {
    throw new Error("unexpected ecdsa signature format");
  }
  if ((first & 0x80) === 0) {
    return { length: first, bytesRead: 1 };
  }

  const count = first & 0x7f;
  if (count === 0 || count > 4) {
    throw new Error("unexpected ecdsa signature format");
  }

  let length = 0;
  for (let i = 0; i < count; i += 1) {
    const next = buffer[offset + 1 + i];
    if (next === undefined) {
      throw new Error("unexpected ecdsa signature format");
    }
    length = (length << 8) | next;
  }

  return { length, bytesRead: 1 + count };
}

function concatFixedLengthIntegers(left, right, size) {
  const normalizedLeft = normalizeUnsignedInteger(left, size);
  const normalizedRight = normalizeUnsignedInteger(right, size);
  const out = new Uint8Array(size * 2);
  out.set(normalizedLeft, 0);
  out.set(normalizedRight, size);
  return out;
}

function normalizeUnsignedInteger(value, size) {
  let out = value;
  while (out.length > 0 && out[0] === 0x00) {
    out = out.slice(1);
  }
  if (out.length > size) {
    throw new Error("ecdsa integer is too large");
  }
  const padded = new Uint8Array(size);
  padded.set(out, size - out.length);
  return padded;
}

function toPEM(spki) {
  const base64 = toBase64(new Uint8Array(spki));
  const lines = [];
  for (let i = 0; i < base64.length; i += 64) {
    lines.push(base64.slice(i, i + 64));
  }
  return `-----BEGIN PUBLIC KEY-----\n${lines.join("\n")}\n-----END PUBLIC KEY-----`;
}

function toBase64(bytes) {
  return b64encode(bytes, "std");
}

function base64UrlEncode(bytes) {
  return toBase64(bytes).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/g, "");
}

function randomUUID() {
  if (typeof crypto.randomUUID === "function") {
    return crypto.randomUUID();
  }
  const bytes = new Uint8Array(16);
  crypto.getRandomValues(bytes);
  return base64UrlEncode(bytes);
}

function trim(value) {
  return String(value || "").trim();
}

function trimSlash(value) {
  return trim(value).replace(/\/+$/, "");
}
