export type Resume = {
  id: string;
  filename: string;
  content_type?: string;
  size?: number;
  created_at?: string;
  status?: "processing" | "ready" | "failed";
};

export type ResumeUploadResponse = {
  message: string;
  resume_id?: string;
};
