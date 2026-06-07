export const API_URL = (import.meta.env.VITE_API_URL as string) || "http://localhost:8000";

export interface Video {
  ID: string;
  Name: string;
  S3Url: string;
  ThumbnailUrl: string;
  Duration: number;
  Resolution: string;
  Size: number;
  Progress: number;
  ViewCount: number;
  Owner: string;
  UploadedAt: string;
  UpdatedAt: string;
}

function getToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("vuetube_token");
}

export function setToken(token: string) {
  localStorage.setItem("vuetube_token", token);
}

export function clearToken() {
  localStorage.removeItem("vuetube_token");
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = getToken();
  const headers: Record<string, string> = {
    ...(options.headers as Record<string, string> | undefined),
  };
  if (!(options.body instanceof FormData) && options.body) {
    headers["Content-Type"] = headers["Content-Type"] || "application/json";
  }
  if (token) headers["Authorization"] = `Bearer ${token}`;

  const res = await fetch(`${API_URL}${path}`, { ...options, headers });
  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(text || `Request failed: ${res.status}`);
  }
  const ct = res.headers.get("content-type") || "";
  if (ct.includes("application/json")) return res.json();
  return (await res.text()) as unknown as T;
}

export const api = {
  signup: (data: { first_name: string; last_name: string; email: string; password: string, country: string; }) =>
    request<{ token: string }>("/auth/signup", { method: "POST", body: JSON.stringify(data) }),
  login: (data: { email: string; password: string }) =>
    request<{ token: string }>("/auth/login", { method: "POST", body: JSON.stringify(data) }),
  getVideo: (id: string) => request<Video>(`/videos/${id}`),
  feed: async (seedId?: string, limit = 20): Promise<Video[]> => {
    if (seedId) {
      const res = await request<{ message: string; recommendations: Video[] }>(
        `/videos/feed/${seedId}?limit=${limit}`
      );
      return res.recommendations ?? [];
    }
    const res = await request<{ message: string; feed: Video[] }>(
      `/feed?limit=${limit}`
    );
    return res.feed ?? [];
  },
  like: (id: string) =>
    request<{ liked: boolean; like_count: number }>(`/videos/${id}/like`, { method: "POST" }),
  upload: async (file: File, title: string, onProgress?: (pct: number) => void) => {
    const fd = new FormData();
    fd.append("video_file", file);
    fd.append("title", title);
    return new Promise<Video>((resolve, reject) => {
      const xhr = new XMLHttpRequest();
      xhr.open("POST", `${API_URL}/videos/upload`);
      const token = getToken();
      if (token) xhr.setRequestHeader("Authorization", `Bearer ${token}`);
      xhr.upload.onprogress = (e) => {
        if (e.lengthComputable && onProgress) onProgress(Math.round((e.loaded / e.total) * 100));
      };
      xhr.onload = () => {
        if (xhr.status >= 200 && xhr.status < 300) {
          try { resolve(JSON.parse(xhr.responseText)); } catch (e) { reject(e); }
        } else reject(new Error(xhr.responseText || `Upload failed: ${xhr.status}`));
      };
      xhr.onerror = () => reject(new Error("Upload failed"));
      xhr.send(fd);
    });
  },
};

export function formatDuration(seconds: number): string {
  if (!seconds || seconds < 0) return "0:00";
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = Math.floor(seconds % 60);
  if (h > 0) return `${h}:${String(m).padStart(2, "0")}:${String(s).padStart(2, "0")}`;
  return `${m}:${String(s).padStart(2, "0")}`;
}

export function formatViews(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
  return `${n ?? 0}`;
}
