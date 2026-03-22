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
