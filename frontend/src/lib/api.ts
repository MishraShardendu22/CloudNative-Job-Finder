import type { RecommendationResponse } from "@/types/job";
import type { Resume, ResumeUploadResponse } from "@/types/resume";
import type { AuthPayload, User } from "@/types/user";

const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_BASE_URL?.replace(/\/$/, "") ?? "/api";

type HttpMethod = "GET" | "POST" | "PUT" | "PATCH" | "DELETE";

type RequestOptions = {
  method?: HttpMethod;
  body?: BodyInit | object;
  headers?: HeadersInit;
};

export class ApiError extends Error {
  status: number;

  constructor(message: string, status: number) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

function getClientTokenFromCookie() {
  if (typeof document === "undefined") return null;

  const tokenPair = document.cookie
    .split(";")
    .map((part) => part.trim())
    .find((part) => part.startsWith("jwt="));

  if (!tokenPair) return null;
  const [, token] = tokenPair.split("=");
  return token ? decodeURIComponent(token) : null;
}

async function safeParse(response: Response) {
  const text = await response.text();
  if (!text) return null;

  try {
    return JSON.parse(text) as unknown;
  } catch {
    return text;
  }
}

async function request<T>(
  path: string,
  options: RequestOptions = {},
): Promise<T> {
  const { method = "GET", body, headers } = options;

  const finalHeaders = new Headers(headers);
  let finalBody: BodyInit | undefined;

  if (body instanceof FormData) {
    finalBody = body;
  } else if (body !== undefined) {
    finalHeaders.set("Content-Type", "application/json");
    finalBody = JSON.stringify(body);
  }

  const token = getClientTokenFromCookie();
  if (token && !finalHeaders.has("Authorization")) {
    finalHeaders.set("Authorization", `Bearer ${token}`);
  }

  const response = await fetch(`${API_BASE_URL}${path}`, {
    method,
    body: finalBody,
    headers: finalHeaders,
    credentials: "include",
    cache: "no-store",
  });

  const parsed = await safeParse(response);

  if (!response.ok) {
    const message =
      typeof parsed === "object" && parsed && "message" in parsed
        ? String(parsed.message)
        : `Request failed with status ${response.status}`;
    throw new ApiError(message, response.status);
  }

  return parsed as T;
}

function normalizeResumesResponse(payload: unknown): Resume[] {
  const rawItems = Array.isArray(payload)
    ? payload
    : payload && typeof payload === "object" && "items" in payload
      ? (payload as { items?: unknown }).items
      : [];

  if (!Array.isArray(rawItems)) return [];

  return rawItems.map((item, index) => {
    const row = (item ?? {}) as {
      id?: string;
      filename?: string;
      file_url?: string;
      created_at?: string;
      status?: "processing" | "ready" | "failed";
      parsed_keywords?: unknown[];
    };

    const fileURL = row.file_url ?? "";
    const derivedName =
      fileURL.split("/").filter(Boolean).pop()?.split("?")[0] ??
      `Resume ${index + 1}`;

    return {
      id: row.id ?? String(index),
      filename: row.filename ?? decodeURIComponent(derivedName),
      created_at: row.created_at,
      status:
        row.status ??
        (Array.isArray(row.parsed_keywords) && row.parsed_keywords.length > 0
          ? "ready"
          : "processing"),
    } satisfies Resume;
  });
}

function normalizeRecommendationsResponse(
  payload: unknown,
): RecommendationResponse {
  const page = (payload ?? {}) as {
    resume_id?: string;
    jobs?: unknown[];
    items?: unknown[];
  };

  const rawJobs = Array.isArray(page.jobs)
    ? page.jobs
    : Array.isArray(page.items)
      ? page.items
      : [];

  return {
    resume_id: page.resume_id ?? "",
    jobs: rawJobs.map((job, index) => {
      const row = (job ?? {}) as {
        id?: string;
        job_id?: string;
        title?: string;
        company?: string;
        location?: string;
        score?: number;
        apply_url?: string;
        url?: string;
        summary?: string;
      };

      return {
        id: row.id ?? row.job_id ?? String(index),
        title: row.title ?? "Untitled role",
        company: row.company ?? "Unknown company",
        location: row.location ?? "Remote",
        score: typeof row.score === "number" ? row.score : 0,
        apply_url: row.apply_url ?? row.url ?? "#",
        summary: row.summary,
      };
    }),
  };
}

export const api = {
  signup: (payload: AuthPayload) =>
    request<{ token?: string; user?: User; message?: string }>("/signup", {
      method: "POST",
      body: payload,
    }),

  login: (payload: AuthPayload) =>
    request<{ token?: string; user?: User; message?: string }>("/login", {
      method: "POST",
      body: payload,
    }),

  profile: () => request<User>("/profile"),

  resumes: async () => {
    const payload = await request<unknown>("/resumes");
    return normalizeResumesResponse(payload);
  },

  uploadResume: (file: File) => {
    const formData = new FormData();
    formData.append("resume", file);
    return request<ResumeUploadResponse>("/resume/upload", {
      method: "POST",
      body: formData,
    });
  },

  recommendations: (resumeId: string) =>
    request<unknown>(`/recommendations/${resumeId}`).then(
      normalizeRecommendationsResponse,
    ),
};
