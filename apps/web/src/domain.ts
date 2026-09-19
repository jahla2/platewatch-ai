export type DetectionEvent = {
  id: string;
  camera_id: string;
  track_id: number;
  plate: string;
  confidence: number;
  flagged: boolean;
  snapshot_url: string;
  plate_crop_url: string;
  detected_at: string;
};
