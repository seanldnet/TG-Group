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

function authenticatedHeaders(): Record<string, string> {
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


export type TelegramGroup = {
  id: string
  bot_id: string
  telegram_chat_id: number
  title: string
  username: string
  chat_type: "group" | "supergroup" | "channel"
  is_forum: boolean
  status: "active" | "left" | "kicked"
  last_seen_at: string
}

export function listGroups() {
  return request<{ items: TelegramGroup[] }>("/api/v1/groups", {
    headers: authenticatedHeaders(),
  })
}


export type TelegramGroupMessageEvent = {
  id: number
  group_id: string
  telegram_message_id: number
  telegram_user_id: number
  sender_username: string
  sender_display_name: string
  message_type: string
  message_text: string
  telegram_sent_at: string | null
  received_at: string
}

export function listGroupEvents(groupID: string, limit = 50) {
  return request<{ items: TelegramGroupMessageEvent[] }>(
    `/api/v1/groups/${encodeURIComponent(groupID)}/events?limit=${limit}`,
    { headers: authenticatedHeaders() },
  )
}

export type TelegramGroupMember = {
  id: number
  group_id: string
  telegram_user_id: number
  username: string
  first_name: string
  last_name: string
  is_bot: boolean
  status: "member" | "administrator" | "creator" | "restricted" | "left" | "kicked"
  is_anonymous: boolean
  can_manage_chat: boolean
  can_delete_messages: boolean
  can_restrict_members: boolean
  can_invite_users: boolean
  can_pin_messages: boolean
  can_manage_topics: boolean
  joined_at: string | null
  last_seen_at: string
}

export function listGroupMembers(groupID: string, limit = 200) {
  return request<{ items: TelegramGroupMember[] }>(
    `/api/v1/groups/${encodeURIComponent(groupID)}/members?limit=${limit}`,
    { headers: authenticatedHeaders() },
  )
}
