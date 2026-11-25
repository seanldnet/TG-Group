export type User = {
  id: string
  tenant_id: string
  email: string
  display_name: string
  role: string
  created_at: string
}

export type AuthSession = {
  token: string
  expires_at: string
  user: User
}

const TOKEN_KEY = "tg-group-session-token"

export function saveToken(token: string) {
  sessionStorage.setItem(TOKEN_KEY, token)
}

export function getToken() {
  return sessionStorage.getItem(TOKEN_KEY)
}

export function clearToken() {
  sessionStorage.removeItem(TOKEN_KEY)
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(path, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init.headers ?? {}),
    },
    cache: "no-store",
  })

  if (!response.ok) {
    const body = await response.json().catch(() => ({})) as { error?: string }
    throw new Error(body.error || `请求失败（${response.status}）`)
  }

  if (response.status === 204) return undefined as T
  return response.json() as Promise<T>
}

export function register(input: {
  email: string
  password: string
  display_name: string
}) {
  return request<AuthSession>("/api/v1/auth/register", {
    method: "POST",
    body: JSON.stringify(input),
  })
}

export function login(input: { email: string; password: string }) {
  return request<AuthSession>("/api/v1/auth/login", {
    method: "POST",
    body: JSON.stringify(input),
  })
}

export function currentUser() {
  const token = getToken()
  return request<User>("/api/v1/auth/me", {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  })
}

export function logout() {
  const token = getToken()
  return request<void>("/api/v1/auth/logout", {
    method: "POST",
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  })
}
