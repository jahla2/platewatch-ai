import { useEffect, useState } from "react";

import { listDetections, subscribeToDetections } from "./api";
import type { DetectionEvent } from "./domain";

export function App() {
  const [events, setEvents] = useState<DetectionEvent[]>([]);
  const [status, setStatus] = useState("Connecting");

  useEffect(() => {
    let active = true;

    void listDetections()
      .then((items) => {
        if (active) {
          setEvents(items);
          setStatus("Live");
        }
      })
      .catch(() => {
        if (active) {
          setStatus("API unavailable");
        }
      });

    const unsubscribe = subscribeToDetections((event) => {
      if (!active) return;
      setStatus("Live");
      setEvents((current) => [event, ...current].slice(0, 50));
    });

    return () => {
      active = false;
      unsubscribe();
    };
  }, []);

  const flaggedCount = events.filter((event) => event.flagged).length;

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
        <span className={`status ${status === "Live" ? "statusLive" : ""}`}>
          {status}
        </span>
      </header>

      <section className="metrics" aria-label="Detection summary">
        <article>
          <span>Recent detections</span>
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

        {events.length === 0 ? (
          <div className="empty">Waiting for the first confirmed plate.</div>
        ) : (
          <div className="eventList">
            {events.map((event) => (
              <article className="eventCard" key={event.id}>
                <div className="eventPrimary">
                  {event.snapshot_url ? (
                    <img
                      className="vehicleThumb"
                      src={event.snapshot_url}
                      alt={`Vehicle detected as ${event.plate}`}
                      loading="lazy"
                    />
                  ) : (
                    <div className="vehicleThumb placeholder">No image</div>
                  )}
                  <div>
                    <span className="plate">{event.plate}</span>
                    <p>
                      {event.camera_id} · Track #{event.track_id}
                    </p>
                    {event.plate_crop_url ? (
                      <img
                        className="plateThumb"
                        src={event.plate_crop_url}
                        alt={`Plate crop ${event.plate}`}
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
        )}
      </section>
    </main>
  );
}
