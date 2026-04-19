"use client";

export type Actor = {
  user_id: string;
  full_name: string;
  username: string;
  email: string;
  level?: number;
  auth_type: string;
  session_id?: string;
  api_key_id?: string;
  roles: string[];
  permissions: string[];
};

type SessionState = {
  accessToken: string;
  actor: Actor | null;
};

type SessionOptions = {
  redirectOnFailure?: boolean;
};

type StartSessionInput = {
  accessToken: string;
  actor?: Partial<Actor> | null;
  username?: string;
  email?: string;
  fullName?: string;
  userID?: string;
};

const sessionStorageKey = "auth:session";

let state: SessionState = {
  accessToken: "",
  actor: null,
};

let bootstrapPromise: Promise<SessionState | null> | null = null;
const listeners = new Set<(next: SessionState) => void>();
let fetchInterceptorInstalled = false;

const authBypassPaths = new Set([
  "/api",
  "/api/livez",
  "/api/readyz",
  "/api/v1/auth/login",
  "/api/v1/auth/register",
  "/api/v1/auth/activate",
  "/api/v1/auth/forgot-password",
  "/api/v1/auth/reset-password",
  "/api/v1/auth/mfa/verify",
  "/api/v1/auth/mfa/send-otp",
]);

export function getAccessToken() {
  return state.accessToken;
}

export function getActor() {
  return state.actor;
}

export function setSession(next: SessionState) {
  state = normalizeSession(next);
  persistSession(state);
  emitSession();
}

export function startSession(input: StartSessionInput) {
  const accessToken = input.accessToken.trim();
  if (accessToken === "") {
    clearSession();
    return;
  }

  const actor = normalizeActor(
    input.actor ?? {
      user_id: input.userID ?? "",
      full_name: input.fullName ?? "",
      username: input.username ?? "",
      email: input.email ?? "",
      auth_type: "password",
      roles: [],
      permissions: [],
    },
  );

  setSession({
    accessToken,
    actor,
  });
}

export function clearSession() {
  state = {
    accessToken: "",
    actor: null,
  };
  persistSession(state);
  emitSession();
}

export function bootstrapSession(
  options: SessionOptions = {},
): Promise<SessionState | null> {
  if (state.accessToken !== "" || state.actor != null) {
    return Promise.resolve(state);
  }
  if (bootstrapPromise != null) {
    return bootstrapPromise;
  }

  bootstrapPromise = Promise.resolve(restorePersistedSession(options)).finally(() => {
    bootstrapPromise = null;
  });

  return bootstrapPromise;
}

export async function refreshSessionFromCookie(
  options: SessionOptions = {},
): Promise<SessionState | null> {
  return bootstrapSession(options);
}

export async function logoutSession() {
  try {
    if (typeof window !== "undefined") {
      await fetch("/api/v1/auth/logout", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        // In case authAwareFetch is not used, explicitly include credentials
        // so the backend can clear the refresh_token cookie.
        credentials: "include",
      });
    }
  } catch (err) {
    console.error("Failed to call logout API:", err);
  } finally {
    clearSession();
  }
}

export function installAuthFetchInterceptor() {
  if (fetchInterceptorInstalled || typeof window === "undefined") {
    return;
  }

  const originalFetch = window.fetch.bind(window);
  window.fetch = async (input: RequestInfo | URL, init?: RequestInit) => {
    return authAwareFetch(originalFetch, input, init);
  };
  fetchInterceptorInstalled = true;
}

export function subscribeSession(listener: (next: SessionState) => void) {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

function emitSession() {
  for (const listener of listeners) {
    listener(state);
  }
}

async function authAwareFetch(
  originalFetch: typeof fetch,
  input: RequestInfo | URL,
  init?: RequestInit,
) {
  const requestURL = resolveRequestURL(input);
  if (shouldBypassAuth(requestURL, init)) {
    return originalFetch(input, init);
  }

  let accessToken = state.accessToken;
  if (accessToken === "") {
    const restored = await bootstrapSession({ redirectOnFailure: true });
    accessToken = restored?.accessToken ?? "";
  }

  const headers = new Headers(init?.headers);
  if (accessToken !== "") {
    headers.set("Authorization", `Bearer ${accessToken}`);
  }

  const response = await originalFetch(input, withDefaultCredentials(init, headers));
  if (response.status !== 401) {
    return response;
  }

  clearSession();
  redirectToSignIn();
  return response;
}

function withDefaultCredentials(init: RequestInit | undefined, headers: Headers): RequestInit {
  return {
    ...init,
    credentials: init?.credentials ?? "include",
    headers,
  };
}

function shouldBypassAuth(requestURL: URL | null, init?: RequestInit) {
  if (typeof window === "undefined") {
    return true;
  }
  if (requestURL == null) {
    return true;
  }
  if (requestURL.origin !== window.location.origin) {
    return true;
  }
  if (!requestURL.pathname.startsWith("/api/")) {
    return true;
  }
  if (authBypassPaths.has(requestURL.pathname)) {
    return true;
  }

  const headers = new Headers(init?.headers);
  return headers.get("X-Skip-Auth") === "1";
}

function resolveRequestURL(input: RequestInfo | URL): URL | null {
  if (typeof window === "undefined") {
    return null;
  }
  if (typeof input === "string") {
    return new URL(input, window.location.origin);
  }
  if (input instanceof URL) {
    return input;
  }
  return new URL(input.url, window.location.origin);
}

function restorePersistedSession(_options: SessionOptions): SessionState | null {
  if (typeof window === "undefined") {
    return null;
  }

  const raw = window.sessionStorage.getItem(sessionStorageKey);
  if (raw == null || raw.trim() == "") {
    return null;
  }

  try {
    const parsed = JSON.parse(raw) as Partial<SessionState>;
    const next = normalizeSession({
      accessToken: typeof parsed.accessToken === "string" ? parsed.accessToken : "",
      actor: parsed.actor != null ? normalizeActor(parsed.actor) : null,
    });
    state = next;
    emitSession();
    return next.accessToken === "" ? null : next;
  } catch {
    clearSession();
    return null;
  }
}

function normalizeSession(next: SessionState): SessionState {
  return {
    accessToken: next.accessToken.trim(),
    actor: next.actor != null ? normalizeActor(next.actor) : null,
  };
}

function normalizeActor(actor: Partial<Actor>): Actor {
  return {
    user_id: typeof actor.user_id === "string" ? actor.user_id : "",
    full_name: typeof actor.full_name === "string" ? actor.full_name : "",
    username: typeof actor.username === "string" ? actor.username : "",
    email: typeof actor.email === "string" ? actor.email : "",
    level: typeof actor.level === "number" ? actor.level : undefined,
    auth_type: typeof actor.auth_type === "string" && actor.auth_type.trim() !== "" ? actor.auth_type : "password",
    session_id: typeof actor.session_id === "string" ? actor.session_id : undefined,
    api_key_id: typeof actor.api_key_id === "string" ? actor.api_key_id : undefined,
    roles: Array.isArray(actor.roles) ? actor.roles.filter((v): v is string => typeof v === "string") : [],
    permissions: Array.isArray(actor.permissions)
      ? actor.permissions.filter((v): v is string => typeof v === "string")
      : [],
  };
}

function persistSession(next: SessionState) {
  if (typeof window === "undefined") {
    return;
  }

  if (next.accessToken === "" && next.actor == null) {
    window.sessionStorage.removeItem(sessionStorageKey);
    return;
  }

  window.sessionStorage.setItem(sessionStorageKey, JSON.stringify(next));
}

function redirectToSignIn() {
  if (typeof window === "undefined") {
    return;
  }
  const currentPath = window.location.pathname;
  if (isAuthPath(currentPath)) {
    return;
  }
  window.location.replace("/signin");
}

function isAuthPath(pathname: string) {
  return (
    pathname === "/signin" ||
    pathname === "/signup" ||
    pathname === "/verify-email" ||
    pathname.startsWith("/reset-password")
  );
}
