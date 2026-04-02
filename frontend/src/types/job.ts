export type JobRecommendation = {
  id: string;
  title: string;
  company: string;
  location: string;
  score: number;
  apply_url: string;
  summary?: string;
};

export type RecommendationResponse = {
  resume_id: string;
  jobs: JobRecommendation[];
};

export type JobInteractionType = "impression" | "click" | "apply";

export type JobInteractionPayload = {
  resume_id: string;
  job_id: string;
  interaction_type: JobInteractionType;
  source: string;
  metadata?: Record<string, unknown>;
};
