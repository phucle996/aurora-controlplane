"use client";

import { buildRefreshSignature } from "./device-binding";
import type { ProfileViewData } from "../user-profile/types";

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
  actor: Actor | null;
  profile: ProfileViewData | null;
};

type SessionOptions = {
  redirectOnFailure?: boolean;
};

type APIEnvelope<T> = {
  data?: T;
  message?: string;
  error?: string;
};

type WhoAmIResponseData = {
  user_id?: string;
  username?: string;
  email?: string;
  phone?: string;
  full_name?: string;
  company?: string;
  referral_source?: string;
  job_function?: string;
  country?: string;
  avatar_url?: string;
  bio?: string;
  status?: string;
  on_boarding?: boolean;
  level?: number;
  auth_type?: string;
  session_id?: string;
  api_key_id?: string;
  roles?: string[];
  permissions?: string[];
};

type StartSessionInput = {
  actor?: Partial<Actor> | null;
  profile?: Partial<ProfileViewData> | null;
  username?: string;
  email?: string;
  fullName?: string;
  userID?: string;
};

const sessionStorageKey = "auth:session";

let state: SessionState = {
  actor: null,
  profile: null,
};

let bootstrapPromise: Promise<SessionState | null> | null = null;
const listeners = new Set<(next: SessionState) => void>();
let fetchInterceptorInstalled = false;
let nativeFetch: typeof fetch | null = null;

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
  "/api/v1/auth/refresh",
]);

export function getAccessToken() {
  return "";
}

export function getActor() {
  return state.actor;
}

export function getProfile() {
  return state.profile;
}

export function setSession(next: SessionState) {
  state = normalizeSession(next);
  persistSession(state);
  emitSession();
}

export function startSession(input: StartSessionInput) {
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

  const profile = normalizeProfile(input.profile ?? null, actor);
  setSession({ actor, profile });
}

export function clearSession() {
  state = { actor: null, profile: null };
  persistSession(state);
  emitSession();
}

export function bootstrapSession(
  options: SessionOptions = {},
): Promise<SessionState | null> {
  if (state.actor != null) {
    return Promise.resolve(state);
  }
  if (bootstrapPromise != null) {
    return bootstrapPromise;
  }

  bootstrapPromise = (async () => {
    const restored = restorePersistedSession(options);
    const hydrated = await hydrateSessionFromWhoAmI(getNativeFetch());
    if (hydrated != null) {
      return hydrated;
    }

    const refreshed = await tryRefreshWithCookies(getNativeFetch());
    if (refreshed != null) {
      return refreshed;
    }

    if (restored != null && restored.actor != null) {
      return restored;
    }

    if (options.redirectOnFailure) {
      clearSession();
      redirectToSignIn();
    }
    return null;
  })().finally(() => {
    bootstrapPromise = null;
  });

  return bootstrapPromise;
}

export async function refreshSessionFromCookie(
  options: SessionOptions = {},
): Promise<SessionState | null> {
  const refreshed = await tryRefreshWithCookies(getNativeFetch());
  if (refreshed != null) {
    return refreshed;
  }
  if (options.redirectOnFailure) {
    clearSession();
    redirectToSignIn();
  }
  return null;
}

export async function hydrateSessionFromWhoAmI(fetchImpl: typeof fetch = getNativeFetch()): Promise<SessionState | null> {
  if (typeof window === "undefined") {
    return null;
  }

  try {
    const response = await fetchImpl("/api/v1/whoami", {
      method: "GET",
      credentials: "include",
      headers: {
        "X-Skip-Auth": "1",
      },
    });
    if (response.status === 401) {
      return null;
    }
    if (!response.ok) {
      return null;
    }

    const result = (await response.json()) as APIEnvelope<WhoAmIResponseData>;
    const actorInput = buildActorFromWhoAmI(result.data);
    if ((actorInput.user_id ?? "").trim() === "") {
      return null;
    }
    const actor = normalizeActor(actorInput);
    const profile = normalizeProfile(buildProfileFromWhoAmI(result.data), actor);
    const next = normalizeSession({ actor, profile });
    setSession(next);
    return next;
  } catch {
    return null;
  }
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

  nativeFetch = window.fetch.bind(window);
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

  const response = await originalFetch(input, withDefaultCredentials(init));
  if (response.status !== 401) {
    return response;
  }

  const refreshed = await tryRefreshWithCookies(originalFetch);
  if (refreshed == null) {
    clearSession();
    redirectToSignIn();
    return response;
  }

  const retried = await originalFetch(input, withDefaultCredentials(init));
  if (retried.status !== 401) {
    return retried;
  }

  clearSession();
  redirectToSignIn();
  return retried;
}

function withDefaultCredentials(init: RequestInit | undefined): RequestInit {
  return {
    ...init,
    credentials: init?.credentials ?? "include",
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
      actor: parsed.actor != null ? normalizeActor(parsed.actor) : null,
      profile: parsed.profile != null ? normalizeProfile(parsed.profile, parsed.actor != null ? normalizeActor(parsed.actor) : null) : null,
    });
    state = next;
    emitSession();
    return next.actor == null || next.profile == null ? null : next;
  } catch {
    clearSession();
    return null;
  }
}

function getNativeFetch(): typeof fetch {
  if (nativeFetch != null) {
    return nativeFetch;
  }
  if (typeof window !== "undefined") {
    nativeFetch = window.fetch.bind(window);
    return nativeFetch;
  }
  return fetch;
}

async function tryRefreshWithCookies(fetchImpl: typeof fetch): Promise<SessionState | null> {
  if (typeof window === "undefined") {
    return null;
  }

  try {
    const signature = await buildRefreshSignature();
    const response = await fetchImpl("/api/v1/auth/refresh", {
      method: "POST",
      credentials: "include",
      headers: {
        "Content-Type": "application/json",
        "X-Skip-Auth": "1",
      },
      body: JSON.stringify(signature),
    });
    if (response.status !== 204) {
      return null;
    }

    return await hydrateSessionFromWhoAmI(fetchImpl);
  } catch {
    return null;
  }
}

function normalizeSession(next: SessionState): SessionState {
  return {
    actor: next.actor != null ? normalizeActor(next.actor) : null,
    profile: normalizeProfile(next.profile, next.actor),
  };
}

function buildActorFromWhoAmI(data: WhoAmIResponseData | undefined): Partial<Actor> {
  return {
    user_id: typeof data?.user_id === "string" ? data.user_id : "",
    full_name: typeof data?.full_name === "string" ? data.full_name : "",
    username: typeof data?.username === "string" ? data.username : "",
    email: typeof data?.email === "string" ? data.email : "",
    level: typeof data?.level === "number" ? data.level : undefined,
    auth_type: typeof data?.auth_type === "string" && data.auth_type.trim() !== "" ? data.auth_type : "password",
    session_id: typeof data?.session_id === "string" ? data.session_id : undefined,
    api_key_id: typeof data?.api_key_id === "string" ? data.api_key_id : undefined,
    roles: Array.isArray(data?.roles) ? data?.roles.filter((v): v is string => typeof v === "string") : [],
    permissions: Array.isArray(data?.permissions)
      ? data?.permissions.filter((v): v is string => typeof v === "string")
      : [],
  };
}

function buildProfileFromWhoAmI(data: WhoAmIResponseData | undefined): Partial<ProfileViewData> | null {
  if (data == null) {
    return null;
  }

  return {
    id: typeof data.user_id === "string" ? data.user_id : "",
    username: typeof data.username === "string" ? data.username : "",
    email: typeof data.email === "string" ? data.email : "",
    status: typeof data.status === "string" ? data.status : "",
    on_boarding: typeof data.on_boarding === "boolean" ? data.on_boarding : false,
    profile: {
      full_name: typeof data.full_name === "string" ? data.full_name : "",
      company: typeof data.company === "string" ? data.company : "",
      referral_source: typeof data.referral_source === "string" ? data.referral_source : "",
      phone: typeof data.phone === "string" ? data.phone : "",
      job_function: typeof data.job_function === "string" ? data.job_function : "",
      country: typeof data.country === "string" ? data.country : "",
      avatar_url: typeof data.avatar_url === "string" ? data.avatar_url : "",
      bio: typeof data.bio === "string" ? data.bio : "",
    },
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

  if (next.actor == null) {
    window.sessionStorage.removeItem(sessionStorageKey);
    return;
  }

  window.sessionStorage.setItem(sessionStorageKey, JSON.stringify(next));
}

function normalizeProfile(profile: Partial<ProfileViewData> | null, actor: Partial<Actor> | null): ProfileViewData | null {
  if (profile == null && actor == null) {
    return null;
  }

  const fallbackFullName = actor?.full_name ?? "";
  const fallbackUsername = actor?.username ?? "";
  const fallbackEmail = actor?.email ?? "";

  return {
    id: typeof profile?.id === "string" && profile.id.trim() !== "" ? profile.id : (actor?.user_id ?? ""),
    username: typeof profile?.username === "string" && profile.username.trim() !== "" ? profile.username : fallbackUsername,
    email: typeof profile?.email === "string" && profile.email.trim() !== "" ? profile.email : fallbackEmail,
    status: typeof profile?.status === "string" ? profile.status : "",
    on_boarding: typeof profile?.on_boarding === "boolean" ? profile.on_boarding : false,
    profile: {
      full_name: typeof profile?.profile?.full_name === "string" && profile.profile.full_name.trim() !== ""
        ? profile.profile.full_name
        : fallbackFullName,
      company: typeof profile?.profile?.company === "string" ? profile.profile.company : "",
      referral_source: typeof profile?.profile?.referral_source === "string" ? profile.profile.referral_source : "",
      phone: typeof profile?.profile?.phone === "string" ? profile.profile.phone : "",
      job_function: typeof profile?.profile?.job_function === "string" ? profile.profile.job_function : "",
      country: typeof profile?.profile?.country === "string" ? profile.profile.country : "",
      avatar_url: typeof profile?.profile?.avatar_url === "string" ? profile.profile.avatar_url : "",
      bio: typeof profile?.profile?.bio === "string" ? profile.profile.bio : "",
    },
  };
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
