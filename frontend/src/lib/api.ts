import type {
  JobInteractionPayload,
  RecommendationResponse,
} from "@/types/job";
import type { Resume, ResumeUploadResponse } from "@/types/resume";
import type { AuthPayload, ProfileUpdatePayload, User } from "@/types/user";

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

function getErrorMessage(parsed: unknown, status: number, statusText: string) {
  if (typeof parsed === "string" && parsed.trim()) {
    return parsed;
  }

  if (parsed && typeof parsed === "object") {
    const candidate = parsed as {
      message?: unknown;
      error?: unknown;
      detail?: unknown;
    };

    if (typeof candidate.message === "string" && candidate.message.trim()) {
      return candidate.message;
    }

    if (typeof candidate.error === "string" && candidate.error.trim()) {
      return candidate.error;
    }

    if (typeof candidate.detail === "string" && candidate.detail.trim()) {
      return candidate.detail;
    }
  }

  if (status === 409) {
    return "Request conflict. The resource may already exist.";
  }

  return statusText
    ? `${statusText} (status ${status})`
    : `Request failed with status ${status}`;
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
    const message = getErrorMessage(
      parsed,
      response.status,
      response.statusText,
    );
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

    const prettyName = derivedName.replace(/^\d{8}T\d{6}_[^_]+_(.+)$/, "$1");

    return {
      id: row.id ?? String(index),
      filename: row.filename ?? decodeURIComponent(prettyName),
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

  const rawScores = rawJobs
    .map((job) => {
      const row = (job ?? {}) as { score?: unknown };
      return typeof row.score === "number" && Number.isFinite(row.score)
        ? row.score
        : 0;
    })
    .filter((score) => score >= 0);

  const maxRawScore = rawScores.length > 0 ? Math.max(...rawScores) : 0;

  const normalizeScore = (raw: number) => {
    if (raw <= 0) return 0;

    if (maxRawScore <= 1) {
      return Math.min(100, raw * 100);
    }

    if (maxRawScore <= 100) {
      return Math.min(100, raw);
    }

    return Math.min(100, (raw / maxRawScore) * 100);
  };

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
        score:
          typeof row.score === "number" && Number.isFinite(row.score)
            ? normalizeScore(row.score)
            : 0,
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

  updateProfile: (payload: ProfileUpdatePayload) =>
    request<User>("/profile", {
      method: "PUT",
      body: payload,
    }),

  resumes: async () => {
    const payload = await request<unknown>("/resumes");
    return normalizeResumesResponse(payload);
  },

  deleteResume: (resumeId: string) =>
    request<{ message?: string }>(`/resumes/${resumeId}`, {
      method: "DELETE",
    }),

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

  trackInteraction: (payload: JobInteractionPayload) =>
    request<{ status: string }>("/interactions", {
      method: "POST",
      body: payload,
    }),
};
