"use client";

const DEVICE_BINDING_DB_NAME = "controlplane-auth-device-binding";
const DEVICE_BINDING_STORE_NAME = "device-binding";
const DEVICE_BINDING_STORE_KEY = "current";
const DEVICE_KEY_ALGORITHM = "ES256";

type StoredDeviceBinding = {
  fingerprint: string;
  publicKeyPem: string;
  algorithm: string;
  privateKey: CryptoKey;
};

export type LoginDeviceBinding = {
  device_fingerprint: string;
  device_public_key: string;
  device_key_algorithm: string;
};

export type RefreshSignature = {
  jti: string;
  iat: number;
  htm: string;
  htu: string;
  token_hash: string;
  device_id: string;
  signature: string;
};

export async function ensureLoginDeviceBinding(): Promise<LoginDeviceBinding> {
  const binding = await getOrCreateDeviceBinding();
  return {
    device_fingerprint: binding.fingerprint,
    device_public_key: binding.publicKeyPem,
    device_key_algorithm: binding.algorithm,
  };
}

export async function buildRefreshSignature(): Promise<RefreshSignature> {
  const binding = await readDeviceBinding();
  if (binding == null) {
    throw new Error("Device binding is unavailable.");
  }

  const deviceID = readCookie("device_id");
  const refreshTokenHash = readCookie("refresh_token_hash");
  if (deviceID.trim() === "" || refreshTokenHash.trim() === "") {
    throw new Error("Session cookies are unavailable.");
  }

  const jti = generateJTI();
  const iat = Math.floor(Date.now() / 1000);
  const htm = "POST";
  const htu = new URL("/api/v1/auth/refresh", window.location.origin).toString();
  const payload = [jti, String(iat), htm, htu, refreshTokenHash, deviceID].join("\n");
  const signature = await signPayload(binding.privateKey, payload);

  return {
    jti,
    iat,
    htm,
    htu,
    token_hash: refreshTokenHash,
    device_id: deviceID,
    signature,
  };
}

async function getOrCreateDeviceBinding(): Promise<StoredDeviceBinding> {
  const existing = await readDeviceBinding();
  if (existing != null) {
    return existing;
  }

  const generated = await createDeviceBinding();
  await writeDeviceBinding(generated);
  return generated;
}

async function createDeviceBinding(): Promise<StoredDeviceBinding> {
  if (typeof window === "undefined" || !window.crypto?.subtle) {
    throw new Error("Device binding is unavailable.");
  }

  const keyPair = await window.crypto.subtle.generateKey(
    {
      name: "ECDSA",
      namedCurve: "P-256",
    },
    true,
    ["sign", "verify"],
  );

  if (!("publicKey" in keyPair) || !("privateKey" in keyPair)) {
    throw new Error("Device binding is unavailable.");
  }

  const publicKeyPem = await exportPublicKeyPem(keyPair.publicKey);

  return {
    fingerprint: generateFingerprint(),
    publicKeyPem,
    algorithm: DEVICE_KEY_ALGORITHM,
    privateKey: keyPair.privateKey,
  };
}

async function readDeviceBinding(): Promise<StoredDeviceBinding | null> {
  if (typeof window === "undefined" || typeof window.indexedDB === "undefined") {
    return null;
  }

  const db = await openDatabase();
  try {
    return await new Promise<StoredDeviceBinding | null>((resolve, reject) => {
      const tx = db.transaction(DEVICE_BINDING_STORE_NAME, "readonly");
      const store = tx.objectStore(DEVICE_BINDING_STORE_NAME);
      const request = store.get(DEVICE_BINDING_STORE_KEY);

      request.onerror = () => {
        reject(request.error ?? new Error("Device binding lookup failed."));
      };
      tx.onerror = () => {
        reject(tx.error ?? new Error("Device binding lookup failed."));
      };
      tx.onabort = () => {
        reject(tx.error ?? new Error("Device binding lookup failed."));
      };
      request.onsuccess = () => {
        resolve((request.result as StoredDeviceBinding | undefined) ?? null);
      };
    });
  } finally {
    db.close();
  }
}

async function writeDeviceBinding(binding: StoredDeviceBinding): Promise<void> {
  if (typeof window === "undefined" || typeof window.indexedDB === "undefined") {
    throw new Error("Device binding is unavailable.");
  }

  const db = await openDatabase();
  try {
    await new Promise<void>((resolve, reject) => {
      const tx = db.transaction(DEVICE_BINDING_STORE_NAME, "readwrite");
      const store = tx.objectStore(DEVICE_BINDING_STORE_NAME);
      const request = store.put(binding, DEVICE_BINDING_STORE_KEY);

      request.onerror = () => {
        reject(request.error ?? new Error("Device binding save failed."));
      };
      tx.onerror = () => {
        reject(tx.error ?? new Error("Device binding save failed."));
      };
      tx.onabort = () => {
        reject(tx.error ?? new Error("Device binding save failed."));
      };
      tx.oncomplete = () => resolve();
    });
  } finally {
    db.close();
  }
}

async function openDatabase(): Promise<IDBDatabase> {
  if (typeof window === "undefined" || typeof window.indexedDB === "undefined") {
    throw new Error("Device binding is unavailable.");
  }

  return await new Promise<IDBDatabase>((resolve, reject) => {
    const request = window.indexedDB.open(DEVICE_BINDING_DB_NAME, 1);

    request.onerror = () => {
      reject(request.error ?? new Error("Device binding database could not be opened."));
    };
    request.onupgradeneeded = () => {
      const db = request.result;
      if (!db.objectStoreNames.contains(DEVICE_BINDING_STORE_NAME)) {
        db.createObjectStore(DEVICE_BINDING_STORE_NAME);
      }
    };
    request.onsuccess = () => resolve(request.result);
  });
}

async function exportPublicKeyPem(publicKey: CryptoKey): Promise<string> {
  const spki = await window.crypto.subtle.exportKey("spki", publicKey);
  const base64 = arrayBufferToBase64(spki);
  const wrapped = wrapPemLines(base64);
  return `-----BEGIN PUBLIC KEY-----\n${wrapped}\n-----END PUBLIC KEY-----`;
}

async function signPayload(privateKey: CryptoKey, payload: string): Promise<string> {
  const signature = await window.crypto.subtle.sign(
    {
      name: "ECDSA",
      hash: "SHA-256",
    },
    privateKey,
    new TextEncoder().encode(payload),
  );

  return arrayBufferToBase64Url(ecdsaSignatureToRaw(new Uint8Array(signature)));
}

function generateFingerprint(): string {
  if (typeof window !== "undefined" && typeof window.crypto?.randomUUID === "function") {
    return window.crypto.randomUUID();
  }

  const bytes = new Uint8Array(16);
  if (typeof window !== "undefined" && window.crypto?.getRandomValues) {
    window.crypto.getRandomValues(bytes);
  }
  return arrayBufferToBase64Url(bytes);
}

function generateJTI(): string {
  if (typeof window !== "undefined" && typeof window.crypto?.randomUUID === "function") {
    return window.crypto.randomUUID();
  }

  const bytes = new Uint8Array(16);
  if (typeof window !== "undefined" && window.crypto?.getRandomValues) {
    window.crypto.getRandomValues(bytes);
  }
  return arrayBufferToBase64Url(bytes);
}

function readCookie(name: string): string {
  if (typeof document === "undefined") {
    return "";
  }

  const prefix = `${name}=`;
  const parts = document.cookie.split("; ");
  for (const part of parts) {
    if (part.startsWith(prefix)) {
      return decodeURIComponent(part.slice(prefix.length));
    }
  }
  return "";
}

function arrayBufferToBase64(buffer: ArrayBuffer | Uint8Array): string {
  const bytes = buffer instanceof Uint8Array ? buffer : new Uint8Array(buffer);
  let binary = "";
  const chunkSize = 0x8000;

  for (let i = 0; i < bytes.length; i += chunkSize) {
    binary += String.fromCharCode(...bytes.subarray(i, i + chunkSize));
  }

  return btoa(binary);
}

function arrayBufferToBase64Url(buffer: ArrayBuffer | Uint8Array): string {
  return arrayBufferToBase64(buffer).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/g, "");
}

function wrapPemLines(base64: string): string {
  const lines: string[] = [];
  for (let i = 0; i < base64.length; i += 64) {
    lines.push(base64.slice(i, i + 64));
  }
  return lines.join("\n");
}

function ecdsaSignatureToRaw(signature: Uint8Array): Uint8Array {
  if (signature.length === 64) {
    return signature;
  }

  let offset = 0;
  if (signature[offset++] !== 0x30) {
    throw new Error("Unexpected ECDSA signature format.");
  }

  const seqLength = readDerLength(signature, offset);
  offset += seqLength.bytesRead;

  if (signature[offset++] !== 0x02) {
    throw new Error("Unexpected ECDSA signature format.");
  }
  const rLength = readDerLength(signature, offset);
  offset += rLength.bytesRead;
  const r = signature.slice(offset, offset + rLength.length);
  offset += rLength.length;

  if (signature[offset++] !== 0x02) {
    throw new Error("Unexpected ECDSA signature format.");
  }
  const sLength = readDerLength(signature, offset);
  offset += sLength.bytesRead;
  const s = signature.slice(offset, offset + sLength.length);
  offset += sLength.length;

  if (offset !== signature.length) {
    throw new Error("Unexpected ECDSA signature format.");
  }
  if (seqLength.length !== signature.length - seqLength.bytesRead - 1) {
    // Keep the parser strict enough to reject malformed inputs.
    throw new Error("Unexpected ECDSA signature format.");
  }

  return concatFixedLengthIntegers(r, s, 32);
}

function readDerLength(buffer: Uint8Array, offset: number): { length: number; bytesRead: number } {
  const first = buffer[offset];
  if (first === undefined) {
    throw new Error("Unexpected ECDSA signature format.");
  }

  if ((first & 0x80) === 0) {
    return { length: first, bytesRead: 1 };
  }

  const count = first & 0x7f;
  if (count === 0 || count > 4) {
    throw new Error("Unexpected ECDSA signature format.");
  }

  let length = 0;
  for (let i = 0; i < count; i += 1) {
    const next = buffer[offset + 1 + i];
    if (next === undefined) {
      throw new Error("Unexpected ECDSA signature format.");
    }
    length = (length << 8) | next;
  }

  return { length, bytesRead: 1 + count };
}

function concatFixedLengthIntegers(left: Uint8Array, right: Uint8Array, size: number): Uint8Array {
  const normalizedLeft = normalizeUnsignedInteger(left, size);
  const normalizedRight = normalizeUnsignedInteger(right, size);
  const out = new Uint8Array(size * 2);
  out.set(normalizedLeft, 0);
  out.set(normalizedRight, size);
  return out;
}

function normalizeUnsignedInteger(bytes: Uint8Array, size: number): Uint8Array {
  let start = 0;
  while (start < bytes.length - size && bytes[start] === 0) {
    start += 1;
  }

  const trimmed = bytes.slice(start);
  if (trimmed.length > size) {
    throw new Error("Unexpected ECDSA signature format.");
  }

  const out = new Uint8Array(size);
  out.set(trimmed, size - trimmed.length);
  return out;
}
