const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1";

interface ApiOptions {
  method?: string;
  body?: unknown;
  orgSlug?: string;
  email?: string;
}

export async function apiRequest<T>(path: string, options: ApiOptions = {}): Promise<T> {
  const { method = "GET", body, orgSlug = "test-org-pro", email = "admin-pro@test.com" } = options;

  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    "X-Test-Org-ID": `${orgSlug}:${email}`,
  };

  const res = await fetch(`${API_BASE}${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });

  const json = await res.json();
  if (!res.ok) {
    throw new Error(`API ${method} ${path} failed: ${res.status} ${JSON.stringify(json)}`);
  }

  return json as T;
}
