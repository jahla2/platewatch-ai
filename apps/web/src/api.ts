import type { DetectionEvent, DetectionPage } from "./domain";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "";
const REQUEST_TIMEOUT_MS = 5_000;
const GET_MAX_ATTEMPTS = 2;

function delay(milliseconds: number): Promise<void> {
  return new Promise((resolve) => window.setTimeout(resolve, milliseconds));
}

async function fetchWithTimeout(
  input: RequestInfo | URL,
  init: RequestInit = {},
): Promise<Response> {
  const controller = new AbortController();
  const timeout = window.setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);

  try {
    return await fetch(input, {
      ...init,
      credentials: "include",
      signal: controller.signal,
    });
  } catch (error) {
    if (error instanceof DOMException && error.name === "AbortError") {
      throw new Error("Request timed out");
    }
    throw error;
  } finally {
    window.clearTimeout(timeout);
  }
}

async function fetchDetectionPage(url: string): Promise<DetectionPage> {
  let lastError: unknown;

  for (let attempt = 1; attempt <= GET_MAX_ATTEMPTS; attempt += 1) {
    try {
      const response = await fetchWithTimeout(url, {
        headers: { Accept: "application/json" },
      });

      if (response.ok) {
        return (await response.json()) as DetectionPage;
      }

      if (response.status === 401) {
        throw new Error("Operator session expired");
      }
      if (response.status < 500 || attempt === GET_MAX_ATTEMPTS) {
        throw new Error(`Failed to load detections (HTTP ${response.status})`);
      }
      lastError = new Error(`Temporary API error (HTTP ${response.status})`);
    } catch (error) {
      lastError = error;
      if (attempt === GET_MAX_ATTEMPTS) {
        break;
      }
    }

    await delay(250 * attempt);
  }

  throw lastError instanceof Error ? lastError : new Error("Failed to load detections");
}

export async function hasOperatorSession(): Promise<boolean> {
  const response = await fetchWithTimeout(`${API_BASE_URL}/api/v1/session`, {
    method: "GET",
    headers: { Accept: "application/json" },
  });

  if (response.status === 401) {
    return false;
  }
  if (!response.ok) {
    throw new Error(`Unable to verify session (HTTP ${response.status})`);
  }
  return true;
}

export async function createOperatorSession(token: string): Promise<void> {
  const response = await fetchWithTimeout(`${API_BASE_URL}/api/v1/session`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ token }),
  });

  if (response.status === 401) {
    throw new Error("Invalid operator token");
  }
  if (response.status === 429) {
    throw new Error("Too many login attempts. Try again later.");
  }
  if (!response.ok) {
    throw new Error(`Unable to create session (HTTP ${response.status})`);
  }
}

export async function deleteOperatorSession(): Promise<void> {
  const response = await fetchWithTimeout(`${API_BASE_URL}/api/v1/session`, {
    method: "DELETE",
  });
  if (!response.ok && response.status !== 401) {
    throw new Error(`Unable to sign out (HTTP ${response.status})`);
  }
}

export function listDetections(cursor?: string): Promise<DetectionPage> {
  const params = new URLSearchParams({ limit: "50" });
  if (cursor) {
    params.set("cursor", cursor);
  }
  return fetchDetectionPage(`${API_BASE_URL}/api/v1/detections?${params.toString()}`);
}

export type StreamState = "connected" | "reconnecting";

export function subscribeToDetections(
  onDetection: (event: DetectionEvent) => void,
  onStateChange: (state: StreamState) => void,
): () => void {
  const stream = new EventSource(
    `${API_BASE_URL}/api/v1/events/stream`,
    { withCredentials: true },
  );

  stream.onopen = () => onStateChange("connected");
  stream.onerror = () => onStateChange("reconnecting");
  stream.addEventListener("detection", (message) => {
    const event = JSON.parse((message as MessageEvent<string>).data) as DetectionEvent;
    onDetection(event);
  });

  return () => stream.close();
}
