import type { DetectionEvent } from "./domain";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080";

export async function listDetections(): Promise<DetectionEvent[]> {
  const response = await fetch(`${API_BASE_URL}/api/v1/detections?limit=50`);
  if (!response.ok) {
    throw new Error("Failed to load detections");
  }
  return response.json() as Promise<DetectionEvent[]>;
}

export function subscribeToDetections(
  onDetection: (event: DetectionEvent) => void,
): () => void {
  const stream = new EventSource(`${API_BASE_URL}/api/v1/events/stream`);
  stream.addEventListener("detection", (message) => {
    const event = JSON.parse((message as MessageEvent<string>).data) as DetectionEvent;
    onDetection(event);
  });
  return () => stream.close();
}
