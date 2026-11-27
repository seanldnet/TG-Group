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
  refresh_token: string
  refresh_expires_at: string
  user: User
}


export type AgentLevel = {
  id: string
  code: string
  name: string
  price_usdt: number
  sort_order: number
  benefits: Record<string, unknown>
  enabled: boolean
}

export type AgentAccount = {
  is_agent: boolean
  status?: string
  current_level: AgentLevel | null
  parent_user_id: string | null
  wallet_balance_usdt: number
  levels: AgentLevel[]
}

export type WalletLedgerEntry = {
  id: number
  entry_type: string
  amount_usdt: number
  balance_after_usdt: number
  reference_type: string
  reference_id: string
  description: string
  created_at: string
}

const TOKEN_KEY = "tg-group-session-token"
const REFRESH_TOKEN_KEY = "tg-group-refresh-token"

export function saveToken(token: string, refreshToken?: string) {
  sessionStorage.setItem(TOKEN_KEY, token)
  if (refreshToken) sessionStorage.setItem(REFRESH_TOKEN_KEY, refreshToken)
}

export function getToken() {
  return sessionStorage.getItem(TOKEN_KEY)
}

export function clearToken() {
  sessionStorage.removeItem(TOKEN_KEY)
  sessionStorage.removeItem(REFRESH_TOKEN_KEY)
}

let refreshInFlight: Promise<boolean> | null = null

async function refreshSession(): Promise<boolean> {
  if (refreshInFlight) return refreshInFlight
  refreshInFlight = performRefresh()
  try { return await refreshInFlight } finally { refreshInFlight = null }
}

async function performRefresh(): Promise<boolean> {
  const refreshToken = sessionStorage.getItem(REFRESH_TOKEN_KEY)
  if (!refreshToken) return false
  try {
    const response = await fetch("/api/v1/auth/refresh", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: refreshToken }),
      cache: "no-store",
    })
    if (!response.ok) {
      clearToken()
      return false
    }
    const session = await response.json() as AuthSession
    saveToken(session.token, session.refresh_token)
    return true
  } catch {
    return false
  }
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const send = () => {
    const headers = new Headers(init.headers)
    if (headers.has("Authorization")) {
      const token = getToken()
      if (token) headers.set("Authorization", `Bearer ${token}`)
    }
    headers.set("Content-Type", "application/json")
    return fetch(path, { ...init, headers, cache: "no-store" })
  }

  let response = await send()
  const requiresAuth = new Headers(init.headers).has("Authorization")
  const refreshable = requiresAuth && !path.endsWith("/auth/refresh") &&
    !path.endsWith("/auth/login") &&
    !path.endsWith("/auth/register")
  if (response.status === 401 && refreshable && sessionStorage.getItem(REFRESH_TOKEN_KEY)) {
    if (await refreshSession()) response = await send()
    else clearToken()
  }

  if (!response.ok) {
    const body = await response.json().catch(() => ({})) as { error?: string }
    throw new Error(body.error || `请求失败（${response.status}）`)
  }

  if (response.status === 204) return undefined as T
  return response.json() as Promise<T>
}

export type CaptchaConfig = { enabled: boolean; endpoint?: string; widget_url?: string }

export function getCaptchaConfig() {
  return request<CaptchaConfig>("/api/v1/security/captcha")
}

export function register(input: {
  email: string
  password: string
  display_name: string
  captcha_token?: string
}) {
  return request<AuthSession>("/api/v1/auth/register", {
    method: "POST",
    body: JSON.stringify(input),
  })
}

export function login(input: { email: string; password: string; captcha_token?: string }) {
  return request<AuthSession>("/api/v1/auth/login", {
    method: "POST",
    body: JSON.stringify(input),
  })
}

export function updateProfile(input: { display_name: string }) {
  return request<User>("/api/v1/auth/profile", {
    method: "PATCH",
    headers: authenticatedHeaders(),
    body: JSON.stringify(input),
  })
}

export function getAgentAccount() {
  return request<AgentAccount>("/api/v1/account/agent", {
    headers: authenticatedHeaders(),
  })
}

export function listAdminAgentLevels() {
  return request<{ items: AgentLevel[] }>("/api/v1/admin/agent-levels", {
    headers: authenticatedHeaders(),
  })
}

export function createAdminAgentLevel(input: {
  code: string
  name: string
  price_usdt: number
  sort_order: number
  benefits: Record<string, unknown>
}) {
  return request<AgentLevel>("/api/v1/admin/agent-levels", {
    method: "POST",
    headers: authenticatedHeaders(),
    body: JSON.stringify(input),
  })
}

export function updateAdminAgentLevel(levelID: string, input: {
  name: string
  price_usdt: number
  sort_order: number
  benefits: Record<string, unknown>
  enabled: boolean
}) {
  return request<AgentLevel>(`/api/v1/admin/agent-levels/${encodeURIComponent(levelID)}`, {
    method: "PATCH",
    headers: authenticatedHeaders(),
    body: JSON.stringify(input),
  })
}

export type AgentUpgradeOrder = {
  id: string
  order_no: string
  from_level_id: string | null
  target_level_id: string
  amount_usdt: number
  payment_method: "wallet" | "usdt" | "epusdt" | "gmpay"
  status: "pending" | "paid" | "failed" | "cancelled"
  paid_at: string | null
  created_at: string
}

export function upgradeAgentWithWallet(targetLevelID: string, idempotencyKey: string) {
  return request<AgentUpgradeOrder>("/api/v1/account/agent/upgrade", {
    method: "POST",
    headers: { ...authenticatedHeaders(), "Idempotency-Key": idempotencyKey },
    body: JSON.stringify({ target_level_id: targetLevelID }),
  })
}

export function listWalletLedger(limit = 20, offset = 0) {
  return request<{ items: WalletLedgerEntry[]; total: number; limit: number; offset: number }>(
    `/api/v1/account/wallet/ledger?limit=${limit}&offset=${offset}`,
    { headers: authenticatedHeaders() },
  )
}

export function currentUser() {
  const token = getToken()
  return request<User>("/api/v1/auth/me", {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  })
}

export function changePassword(input: { current_password: string; new_password: string }) {
  return request<AuthSession>("/api/v1/auth/password/change", {
    method: "POST",
    headers: authenticatedHeaders(),
    body: JSON.stringify(input),
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

export function listGroupMembers(groupID: string, limit = 20, offset = 0, search = "") {
  const params = new URLSearchParams({ limit: String(limit), offset: String(offset) })
  if (search.trim()) params.set("search", search.trim())
  return request<{ items: TelegramGroupMember[]; total: number; limit: number; offset: number; search?: string }>(
    `/api/v1/groups/${encodeURIComponent(groupID)}/members?${params.toString()}`,
    { headers: authenticatedHeaders() },
  )
}

export type TelegramGroupBotPermissions = {
  group_id: string
  bot_id: string
  bot_status: string
  can_manage_chat: boolean
  can_delete_messages: boolean
  can_restrict_members: boolean
  can_invite_users: boolean
  can_pin_messages: boolean
  can_manage_topics: boolean
}

export function getGroupBotPermissions(groupID: string) {
  return request<TelegramGroupBotPermissions>(
    `/api/v1/groups/${encodeURIComponent(groupID)}/permissions`,
    { headers: authenticatedHeaders() },
  )
}


export type TelegramGroupMemberListEntry = {
  telegram_user_id: number
  list_type: "blacklist" | "whitelist" | "exempt"
  reason: string
  actor_telegram_user_id: number
  created_at: string
}

export function listGroupMemberLists(groupID: string) {
  return request<{ items: TelegramGroupMemberListEntry[] }>(
    `/api/v1/groups/${encodeURIComponent(groupID)}/member-lists`,
    { headers: authenticatedHeaders() },
  )
}

export function upsertGroupMemberList(groupID: string, input: { telegram_user_id: number; list_type: TelegramGroupMemberListEntry["list_type"]; reason: string }) {
  return request<{ ok: boolean }>(`/api/v1/groups/${encodeURIComponent(groupID)}/member-lists`, {
    method: "POST",
    headers: authenticatedHeaders(),
    body: JSON.stringify(input),
  })
}

export function deleteGroupMemberList(groupID: string, userID: number, listType: TelegramGroupMemberListEntry["list_type"]) {
  return request<void>(
    `/api/v1/groups/${encodeURIComponent(groupID)}/member-lists/${userID}?list_type=${encodeURIComponent(listType)}`,
    { method: "DELETE", headers: authenticatedHeaders() },
  )
}

export type TelegramGroupMemberWarning = {
  id: number
  telegram_user_id: number
  actor_telegram_user_id: number
  reason: string
  source: "manual" | "automatic"
  created_at: string
}

export function listGroupMemberWarnings(groupID: string, userID: number, limit = 10, offset = 0) {
  const params = new URLSearchParams({ limit: String(limit), offset: String(offset) })
  return request<{ items: TelegramGroupMemberWarning[]; total: number; limit: number; offset: number }>(
    `/api/v1/groups/${encodeURIComponent(groupID)}/members/${userID}/warnings?${params.toString()}`,
    { headers: authenticatedHeaders() },
  )
}

export function addGroupMemberWarning(groupID: string, userID: number, reason: string) {
  return request<TelegramGroupMemberWarning>(
    `/api/v1/groups/${encodeURIComponent(groupID)}/members/${userID}/warnings`,
    { method: "POST", headers: authenticatedHeaders(), body: JSON.stringify({ reason, source: "manual" }) },
  )
}

export function clearGroupMemberWarnings(groupID: string, userID: number) {
  return request<{ ok: boolean; deleted: number }>(
    `/api/v1/groups/${encodeURIComponent(groupID)}/members/${userID}/warnings`,
    { method: "DELETE", headers: authenticatedHeaders() },
  )
}

export type BulkMemberModerationResult = {
  action: "ban" | "unban" | "mute" | "unmute"
  total: number
  succeeded: number
  failed: number
  items: Array<{ telegram_user_id: number; ok: boolean; error?: string }>
}

export function bulkModerateGroupMembers(
  groupID: string,
  userIDs: number[],
  action: "ban" | "unban" | "mute" | "unmute",
  durationMinutes = 0,
) {
  return request<BulkMemberModerationResult>(`/api/v1/groups/${encodeURIComponent(groupID)}/members/actions/bulk`, {
    method: "POST",
    headers: authenticatedHeaders(),
    body: JSON.stringify({ user_ids: userIDs, action, duration_minutes: durationMinutes }),
  })
}

export function moderateGroupMember(
  groupID: string,
  userID: number,
  action: "ban" | "unban" | "mute" | "unmute",
  durationMinutes = 0,
) {
  return request<{ ok: boolean; action: string; telegram_user_id: number }>(
    `/api/v1/groups/${encodeURIComponent(groupID)}/members/${userID}/action`,
    {
      method: "POST",
      headers: authenticatedHeaders(),
      body: JSON.stringify({ action, duration_minutes: durationMinutes }),
    },
  )
}


export type TelegramGroupRule = {
  id: string
  group_id: string
  rule_type: "keyword_filter" | "sensitive_word" | "anti_spam" | "anti_ad" | "welcome" | "leave_notice" | "join_verification" | "qa" | "warning_escalation" | "media_restriction" | "anti_bot" | "anti_invite_flood"
  name: string
  enabled: boolean
  priority: number
  config: Record<string, unknown>
  created_at: string
  updated_at: string
}

export type TelegramGroupRuleInput = Omit<TelegramGroupRule, "id" | "group_id" | "created_at" | "updated_at">

export function listGroupRules(groupID: string) {
  return request<{ items: TelegramGroupRule[] }>(
    `/api/v1/groups/${encodeURIComponent(groupID)}/rules`,
    { headers: authenticatedHeaders() },
  )
}

export function createGroupRule(groupID: string, input: TelegramGroupRuleInput) {
  return request<TelegramGroupRule>(`/api/v1/groups/${encodeURIComponent(groupID)}/rules`, {
    method: "POST",
    headers: authenticatedHeaders(),
    body: JSON.stringify(input),
  })
}

export function updateGroupRule(groupID: string, ruleID: string, input: TelegramGroupRuleInput) {
  return request<TelegramGroupRule>(
    `/api/v1/groups/${encodeURIComponent(groupID)}/rules/${encodeURIComponent(ruleID)}`,
    { method: "PUT", headers: authenticatedHeaders(), body: JSON.stringify(input) },
  )
}

export function deleteGroupRule(groupID: string, ruleID: string) {
  return request<void>(
    `/api/v1/groups/${encodeURIComponent(groupID)}/rules/${encodeURIComponent(ruleID)}`,
    { method: "DELETE", headers: authenticatedHeaders() },
  )
}


export type TelegramGroupRuleEvent = {
  id: number
  group_id: string
  rule_id: string
  telegram_update_id: number
  telegram_message_id: number
  telegram_user_id: number
  action: string
  status: "succeeded" | "skipped" | "failed"
  detail: string
  created_at: string
}

export function listGroupRuleEvents(groupID: string, limit = 50) {
  return request<{ items: TelegramGroupRuleEvent[] }>(
    `/api/v1/groups/${encodeURIComponent(groupID)}/rule-events?limit=${limit}`,
    { headers: authenticatedHeaders() },
  )
}


export type TelegramIdentity = {
  id: number | null
  username: string
  first_name: string
  last_name: string
}

export type TelegramIdentityStatus = {
  linked: boolean
  telegram: TelegramIdentity
}

export function getTelegramIdentity() {
  return request<TelegramIdentityStatus>("/api/v1/auth/telegram", {
    headers: authenticatedHeaders(),
  })
}

export function createTelegramLink() {
  return request<{ url: string; expires_at: string }>("/api/v1/auth/telegram/link", {
    method: "POST",
    headers: authenticatedHeaders(),
  })
}

export function unlinkTelegramIdentity() {
  return request<void>("/api/v1/auth/telegram", {
    method: "DELETE",
    headers: authenticatedHeaders(),
  })
}

export type AccountSecurityEvent = {
  id: number
  action: string
  resource_type: string
  resource_id: string
  ip_address: string
  created_at: string
}

export function listSecurityEvents(limit = 20) {
  return request<{ items: AccountSecurityEvent[] }>(
    `/api/v1/auth/security-events?limit=${limit}`,
    { headers: authenticatedHeaders() },
  )
}

export type AccountSession = {
  id: string
  user_agent: string
  ip_address: string
  created_at: string
  expires_at: string
  revoked_at: string | null
  current: boolean
}

export function listAccountSessions(limit = 20) {
  return request<{ items: AccountSession[] }>(
    `/api/v1/auth/sessions?limit=${limit}`,
    { headers: authenticatedHeaders() },
  )
}

export function revokeAccountSession(sessionID: string) {
  return request<void>(`/api/v1/auth/sessions/${encodeURIComponent(sessionID)}`, {
    method: "DELETE",
    headers: authenticatedHeaders(),
  })
}


export type PlatformTelegramBotStatus = {
  configured: boolean
  username: string
  running: boolean
  updated_at?: string
}

export function getPlatformTelegramBotStatus() {
  return request<PlatformTelegramBotStatus>("/api/v1/admin/platform/telegram-auth-bot", {
    headers: authenticatedHeaders(),
  })
}

export function savePlatformTelegramBot(input: { token: string; username: string }) {
  return request<PlatformTelegramBotStatus>("/api/v1/admin/platform/telegram-auth-bot", {
    method: "PUT",
    headers: authenticatedHeaders(),
    body: JSON.stringify(input),
  })
}

export function clearPlatformTelegramBot() {
  return request<PlatformTelegramBotStatus>("/api/v1/admin/platform/telegram-auth-bot", {
    method: "DELETE",
    headers: authenticatedHeaders(),
  })
}

export type AdminUser = {
  id: string
  tenant_id: string
  tenant_name: string
  tenant_status: string
  email: string
  display_name: string
  role: "user" | "agent" | "admin"
  status: "active" | "locked" | "deleted"
  telegram_id: number | null
  telegram_username: string
  telegram_first_name: string
  telegram_last_name: string
  bot_count: number
  created_at: string
}

export function listAdminUsers(search = "", limit = 25, offset = 0) {
  const params = new URLSearchParams({ search, limit: String(limit), offset: String(offset) })
  return request<{ items: AdminUser[]; total: number; limit: number; offset: number }>(
    `/api/v1/admin/users?${params.toString()}`,
    { headers: authenticatedHeaders() },
  )
}

export function setAdminUserStatus(userID: string, status: "active" | "locked") {
  return request<{ ok: boolean; status: string }>(
    `/api/v1/admin/users/${encodeURIComponent(userID)}/status`,
    { method: "PATCH", headers: authenticatedHeaders(), body: JSON.stringify({ status }) },
  )
}


export type TelegramWarningEscalationExecution = {
  id: number
  group_id: string
  rule_id: string
  telegram_user_id: number
  warning_count: number
  action: "mute" | "ban"
  duration_minutes: number
  status: "processing" | "succeeded" | "failed" | "skipped"
  detail: string
  created_at: string
  updated_at: string
}

export function listWarningEscalations(groupID: string, userID: number, limit = 10, offset = 0) {
  const params = new URLSearchParams({ limit: String(limit), offset: String(offset) })
  return request<{ items: TelegramWarningEscalationExecution[]; total: number; limit: number; offset: number }>(
    `/api/v1/groups/${encodeURIComponent(groupID)}/members/${userID}/warning-escalations?${params.toString()}`,
    { headers: authenticatedHeaders() },
  )
}

export function retryWarningEscalation(groupID: string, userID: number, executionID: number) {
  return request<TelegramWarningEscalationExecution>(
    `/api/v1/groups/${encodeURIComponent(groupID)}/members/${userID}/warning-escalations/${executionID}/retry`,
    { method: "POST", headers: authenticatedHeaders() },
  )
}
