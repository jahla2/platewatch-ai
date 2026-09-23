import { useCallback, useEffect, useState } from "react";

import { listDetections, subscribeToDetections } from "./api";
import type { DetectionEvent } from "./domain";

type LoadState = "loading" | "success" | "error";

function mergeUnique(
  current: DetectionEvent[],
  incoming: DetectionEvent[],
): DetectionEvent[] {
  const seen = new Set<string>();
  const merged: DetectionEvent[] = [];

  for (const event of [...current, ...incoming]) {
    if (seen.has(event.id)) continue;
    seen.add(event.id);
    merged.push(event);
  }
  return merged;
}

export function App() {
  const [events, setEvents] = useState<DetectionEvent[]>([]);
  const [loadState, setLoadState] = useState<LoadState>("loading");
  const [streamStatus, setStreamStatus] = useState("Connecting");
  const [errorMessage, setErrorMessage] = useState("");
  const [nextCursor, setNextCursor] = useState<string | undefined>();
  const [loadingMore, setLoadingMore] = useState(false);

  const loadInitial = useCallback(async () => {
    setLoadState("loading");
    setErrorMessage("");

    try {
      const page = await listDetections();
      setEvents(page.items);
      setNextCursor(page.next_cursor);
      setLoadState("success");
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : "Failed to load detections");
      setLoadState("error");
    }
  }, []);

  useEffect(() => {
    void loadInitial();

    const unsubscribe = subscribeToDetections(
      (event) => {
        setEvents((current) => mergeUnique([event], current));
      },
      (state) => {
        setStreamStatus(state === "connected" ? "Live" : "Reconnecting");
      },
    );

    return unsubscribe;
  }, [loadInitial]);

  const loadMore = async () => {
    if (!nextCursor || loadingMore) return;

    setLoadingMore(true);
    setErrorMessage("");
    try {
      const page = await listDetections(nextCursor);
      setEvents((current) => mergeUnique(current, page.items));
      setNextCursor(page.next_cursor);
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : "Failed to load older detections");
    } finally {
      setLoadingMore(false);
    }
  };

  const flaggedCount = events.filter((event) => event.flagged).length;
  const hasEvents = events.length > 0;

  return (
    <main className="shell">
      <header className="hero">
        <div>
          <p className="eyebrow">Realtime vehicle intelligence</p>
          <h1>PlateWatch</h1>
          <p className="subtitle">
            Confirmed plate detections with evidence from the live vision pipeline.
          </p>
        </div>
        <span className={`status ${streamStatus === "Live" ? "statusLive" : ""}`}>
          {streamStatus}
        </span>
      </header>

      <section className="metrics" aria-label="Detection summary">
        <article>
          <span>Loaded detections</span>
          <strong>{events.length}</strong>
        </article>
        <article>
          <span>Flagged</span>
          <strong>{flaggedCount}</strong>
        </article>
      </section>

      <section className="panel">
        <div className="panelHeader">
          <div>
            <p className="eyebrow">Event stream</p>
            <h2>Latest plates</h2>
          </div>
        </div>

        {loadState === "loading" && !hasEvents ? (
          <div className="statePanel" role="status">
            <span className="spinner" aria-hidden="true" />
            Loading detections…
          </div>
        ) : null}

        {loadState === "error" && !hasEvents ? (
          <div className="statePanel stateError" role="alert">
            <strong>Could not load detections.</strong>
            <span>{errorMessage}</span>
            <button type="button" onClick={() => void loadInitial()}>
              Retry
            </button>
          </div>
        ) : null}

        {loadState === "success" && !hasEvents ? (
          <div className="empty">
            No confirmed plates yet. The dashboard will update automatically.
          </div>
        ) : null}

        {hasEvents ? (
          <>
            {errorMessage ? (
              <div className="inlineError" role="alert">
                {errorMessage}
              </div>
            ) : null}

            <div className="eventList">
              {events.map((event) => (
                <article className="eventCard" key={event.id}>
                  <div className="eventPrimary">
                    {event.snapshot_url ? (
                      <img
                        className="vehicleThumb"
                        src={event.snapshot_url}
                        alt={`Vehicle detected as ${event.plate_text}`}
                        loading="lazy"
                      />
                    ) : (
                      <div className="vehicleThumb placeholder">No image</div>
                    )}
                    <div>
                      <span className="plate">{event.plate_text}</span>
                      <p className="plateKey">Match key: {event.plate_key}</p>
                      <p>
                        {event.camera_id} · Track #{event.track_id}
                      </p>
                      {event.plate_crop_url ? (
                        <img
                          className="plateThumb"
                          src={event.plate_crop_url}
                          alt={`Plate crop ${event.plate_text}`}
                          loading="lazy"
                        />
                      ) : null}
                    </div>
                  </div>
                  <div className="eventMeta">
                    <span className={event.flagged ? "flagged" : "clear"}>
                      {event.flagged ? "FLAGGED" : "CLEAR"}
                    </span>
                    <strong>{Math.round(event.confidence * 100)}%</strong>
                    <time>{new Date(event.detected_at).toLocaleTimeString()}</time>
                  </div>
                </article>
              ))}
            </div>

            {nextCursor ? (
              <div className="loadMore">
                <button type="button" disabled={loadingMore} onClick={() => void loadMore()}>
                  {loadingMore ? "Loading…" : "Load older detections"}
                </button>
              </div>
            ) : null}
          </>
        ) : null}
      </section>
    </main>
  );
}
