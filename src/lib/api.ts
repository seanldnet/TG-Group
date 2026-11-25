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

export type TelegramBot = {
  id: string
  telegram_bot_id: number
  username: string
  first_name: string
  status: "active" | "disabled"
  verified_at: string
  created_at: string
}

function authenticatedHeaders() {
  const token = getToken()
  return token ? { Authorization: `Bearer ${token}` } : {}
}

export async function listBots() {
  return request<{ items: TelegramBot[] }>("/api/v1/bots", {
    headers: authenticatedHeaders(),
  })
}

export function addBot(token: string) {
  return request<TelegramBot>("/api/v1/bots", {
    method: "POST",
    headers: authenticatedHeaders(),
    body: JSON.stringify({ token }),
  })
}

export function setBotStatus(botID: string, status: TelegramBot["status"]) {
  return request<TelegramBot>(`/api/v1/bots/${encodeURIComponent(botID)}/status`, {
    method: "PATCH",
    headers: authenticatedHeaders(),
    body: JSON.stringify({ status }),
  })
}

export function rotateBotToken(botID: string, token: string) {
  return request<TelegramBot>(`/api/v1/bots/${encodeURIComponent(botID)}/token`, {
    method: "PUT",
    headers: authenticatedHeaders(),
    body: JSON.stringify({ token }),
  })
}

export function removeBot(botID: string) {
  return request<void>(`/api/v1/bots/${encodeURIComponent(botID)}`, {
    method: "DELETE",
    headers: authenticatedHeaders(),
  })
}
