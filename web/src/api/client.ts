export class ApiError extends Error {
  public readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

interface CsrfToken {
  token: string;
  headerName: string;
}

const SAFE_METHODS = new Set(["GET", "HEAD", "OPTIONS"]);

async function fetchCsrf(): Promise<CsrfToken> {
  const response = await fetch("/api/auth/csrf", {
    credentials: "same-origin",
    cache: "no-store",
  });
  if (!response.ok) {
    throw new ApiError(response.status, "登录状态已失效");
  }

  return (await response.json()) as CsrfToken;
}

export async function apiFetch<T>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
  const headers = new Headers(options.headers);
  const method = (options.method ?? "GET").toUpperCase();

  if (!SAFE_METHODS.has(method)) {
    const csrf = await fetchCsrf();

    headers.set(csrf.headerName, csrf.token);
  }

  if (
    options.body &&
    !(options.body instanceof FormData) &&
    !headers.has("Content-Type")
  ) {
    headers.set("Content-Type", "application/json");
  }

  const response = await fetch(path, {
    credentials: "same-origin",
    ...options,
    headers,
  });

  if (!response.ok) {
    let message = `请求失败（${response.status}）`;

    try {
      const body = (await response.json()) as { message?: string };

      if (body.message) {
        message = body.message;
      }
    } catch {
      // non-JSON error body
    }

    throw new ApiError(response.status, message);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return (await response.json()) as T;
}
