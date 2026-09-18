export type DetectionEvent = {
  id: string;
  camera_id: string;
  track_id: number;
  plate: string;
  confidence: number;
  flagged: boolean;
  detected_at: string;
};
