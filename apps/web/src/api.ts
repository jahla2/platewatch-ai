import type { DetectionEvent, DetectionPage } from "./domain";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "";
const REQUEST_TIMEOUT_MS = 5_000;
const GET_MAX_ATTEMPTS = 2;

function delay(milliseconds: number): Promise<void> {
  return new Promise((resolve) => window.setTimeout(resolve, milliseconds));
}

async function fetchDetectionPage(url: string): Promise<DetectionPage> {
  let lastError: unknown;

  for (let attempt = 1; attempt <= GET_MAX_ATTEMPTS; attempt += 1) {
    const controller = new AbortController();
    const timeout = window.setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);

    try {
      const response = await fetch(url, {
        signal: controller.signal,
        headers: { Accept: "application/json" },
      });

      if (response.ok) {
        return (await response.json()) as DetectionPage;
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
    } finally {
      window.clearTimeout(timeout);
    }

    await delay(250 * attempt);
  }

  if (lastError instanceof DOMException && lastError.name === "AbortError") {
    throw new Error("Detection request timed out");
  }
  throw lastError instanceof Error ? lastError : new Error("Failed to load detections");
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
  const stream = new EventSource(`${API_BASE_URL}/api/v1/events/stream`);

  stream.onopen = () => onStateChange("connected");
  stream.onerror = () => onStateChange("reconnecting");
  stream.addEventListener("detection", (message) => {
    const event = JSON.parse((message as MessageEvent<string>).data) as DetectionEvent;
    onDetection(event);
  });

  return () => stream.close();
}
