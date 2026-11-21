package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/seanldnet/tg-group/backend/internal/store"
)

type InstallHandler struct {
	ConfigFile string
	mu         sync.Mutex
}

type installRequest struct {
	PostgresHost     string `json:"postgresHost"`
	PostgresPort     int    `json:"postgresPort"`
	PostgresDatabase string `json:"postgresDatabase"`
	PostgresUsername string `json:"postgresUsername"`
	PostgresPassword string `json:"postgresPassword"`
	PostgresSSLMode  string `json:"postgresSSLMode"`
	RedisAddress     string `json:"redisAddress"`
	RedisPassword    string `json:"redisPassword"`
	RedisDB          int    `json:"redisDB"`
	OwnerUsername    string `json:"ownerUsername"`
	OwnerPassword    string `json:"ownerPassword"`
}

type persistedInstallConfig struct {
	DatabaseURL  string `json:"databaseURL"`
	RedisAddr    string `json:"redisAddr"`
	RedisPassword string `json:"redisPassword"`
	RedisDB      int    `json:"redisDB"`
}

func (h *InstallHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /install", h.page)
	mux.HandleFunc("GET /", h.page)
	mux.HandleFunc("POST /api/install/test", h.testConnections)
	mux.HandleFunc("POST /api/install/complete", h.complete)
}

func (h *InstallHandler) page(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && r.URL.Path != "/install" {
		http.NotFound(w, r)
		return
	}
	if h.isInstalled() {
		http.Error(w, "Installation is already locked. Restart the application to enter the management system.", http.StatusGone)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(installPage))
}

func (h *InstallHandler) testConnections(w http.ResponseWriter, r *http.Request) {
	if h.isInstalled() {
		apiError(w, http.StatusGone, "installation is already locked")
		return
	}
	input, ok := h.decodeAndValidate(w, r, false)
	if !ok {
		return
	}
	if err := testInstallDependencies(r.Context(), input); err != nil {
		apiError(w, http.StatusBadGateway, err.Error())
		return
	}
	apiOK(w, http.StatusOK, map[string]bool{"postgres": true, "redis": true})
}

func (h *InstallHandler) complete(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.isInstalled() {
		apiError(w, http.StatusConflict, "installation is already locked")
		return
	}
	input, ok := h.decodeAndValidate(w, r, true)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	databaseURL, err := buildDatabaseURL(input)
	if err != nil {
		apiError(w, http.StatusBadRequest, "invalid PostgreSQL connection settings")
		return
	}
	db, err := store.OpenPostgres(ctx, databaseURL)
	if err != nil {
		apiError(w, http.StatusBadGateway, "could not connect to PostgreSQL; check host, port, credentials and network access")
		return
	}
	defer db.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: input.RedisAddress, Password: input.RedisPassword, DB: input.RedisDB,
		DialTimeout: 5 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second,
	})
	defer redisClient.Close()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		apiError(w, http.StatusBadGateway, "could not connect to Redis; check address, password and network access")
		return
	}
	if err := store.Migrate(ctx, db); err != nil {
		apiError(w, http.StatusInternalServerError, "database migration failed; inspect server logs and retry")
		return
	}
	if err := store.SeedOwner(ctx, db, strings.TrimSpace(input.OwnerUsername), input.OwnerPassword); err != nil {
		apiError(w, http.StatusInternalServerError, "could not initialize the platform owner")
		return
	}
	config := persistedInstallConfig{
		DatabaseURL: databaseURL, RedisAddr: input.RedisAddress,
		RedisPassword: input.RedisPassword, RedisDB: input.RedisDB,
	}
	if err := h.persistConfig(config); err != nil {
		apiError(w, http.StatusInternalServerError, "database initialized but configuration could not be saved; fix the config directory permissions and retry")
		return
	}
	apiOK(w, http.StatusOK, map[string]any{
		"installed": true,
		"restartRequired": true,
		"message": "Installation completed. Restart the TG Group service to open the management system.",
	})
}

func (h *InstallHandler) decodeAndValidate(w http.ResponseWriter, r *http.Request, requireOwner bool) (installRequest, bool) {
	var input installRequest
	if origin := r.Header.Get("Origin"); origin != "" {
		parsed, err := url.Parse(origin)
		if err != nil || parsed.Host == "" || !strings.EqualFold(parsed.Host, r.Host) {
			apiError(w, http.StatusForbidden, "cross-origin installation requests are not allowed")
			return input, false
		}
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		apiError(w, http.StatusBadRequest, "invalid installation request")
		return input, false
	}
	input.PostgresHost = strings.TrimSpace(input.PostgresHost)
	input.PostgresDatabase = strings.TrimSpace(input.PostgresDatabase)
	input.PostgresUsername = strings.TrimSpace(input.PostgresUsername)
	input.PostgresSSLMode = strings.TrimSpace(input.PostgresSSLMode)
	input.RedisAddress = strings.TrimSpace(input.RedisAddress)
	input.OwnerUsername = strings.TrimSpace(input.OwnerUsername)
	if input.PostgresHost == "" || strings.ContainsAny(input.PostgresHost, " /\\\t\r\n") ||
		input.PostgresPort < 1 || input.PostgresPort > 65535 ||
		input.PostgresDatabase == "" || strings.ContainsAny(input.PostgresDatabase, "/?#") || input.PostgresUsername == "" ||
		input.RedisAddress == "" || strings.ContainsAny(input.RedisAddress, "\r\n") ||
		input.RedisDB < 0 {
		apiError(w, http.StatusBadRequest, "PostgreSQL and Redis connection fields are invalid")
		return input, false
	}
	if input.PostgresSSLMode == "" {
		input.PostgresSSLMode = "prefer"
	}
	switch input.PostgresSSLMode {
	case "disable", "allow", "prefer", "require", "verify-ca", "verify-full":
	default:
		apiError(w, http.StatusBadRequest, "unsupported PostgreSQL SSL mode")
		return input, false
	}
	if requireOwner {
		if input.OwnerUsername == "" || len(input.OwnerUsername) > 64 ||
			len(input.OwnerPassword) < 12 || len(input.OwnerPassword) > 72 {
			apiError(w, http.StatusBadRequest, "set an owner username and a password between 12 and 72 bytes")
			return input, false
		}
	}
	redisHost, redisPort, err := net.SplitHostPort(input.RedisAddress)
	if err != nil || redisHost == "" {
		apiError(w, http.StatusBadRequest, "Redis address must include host and port, for example 127.0.0.1:6379")
		return input, false
	}
	portNumber, err := strconv.Atoi(redisPort)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		apiError(w, http.StatusBadRequest, "Redis port must be between 1 and 65535")
		return input, false
	}
	return input, true
}

func buildDatabaseURL(input installRequest) (string, error) {
	host := net.JoinHostPort(input.PostgresHost, strconv.Itoa(input.PostgresPort))
	u := &url.URL{
		Scheme: "postgres",
		User: url.UserPassword(input.PostgresUsername, input.PostgresPassword),
		Host: host,
		Path: "/" + input.PostgresDatabase,
	}
	query := url.Values{}
	query.Set("sslmode", input.PostgresSSLMode)
	u.RawQuery = query.Encode()
	return u.String(), nil
}

func testInstallDependencies(parent context.Context, input installRequest) error {
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	databaseURL, err := buildDatabaseURL(input)
	if err != nil {
		return errors.New("invalid PostgreSQL connection settings")
	}
	db, err := store.OpenPostgres(ctx, databaseURL)
	if err != nil {
		return errors.New("PostgreSQL connection failed")
	}
	defer db.Close()
	client := redis.NewClient(&redis.Options{Addr: input.RedisAddress, Password: input.RedisPassword, DB: input.RedisDB, DialTimeout: 5 * time.Second})
	defer client.Close()
	if err := client.Ping(ctx).Err(); err != nil {
		return errors.New("Redis connection failed")
	}
	return nil
}

func (h *InstallHandler) isInstalled() bool {
	_, err := os.Lstat(h.ConfigFile)
	return err == nil || !os.IsNotExist(err)
}

func (h *InstallHandler) persistConfig(config persistedInstallConfig) error {
	if h.ConfigFile == "" {
		return errors.New("APP_CONFIG_FILE is empty")
	}
	if err := os.MkdirAll(filepath.Dir(h.ConfigFile), 0750); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	if h.isInstalled() {
		return errors.New("installation config already exists")
	}
	file, err := os.CreateTemp(filepath.Dir(h.ConfigFile), ".tg-group-config-*")
	if err != nil {
		return fmt.Errorf("create temporary config: %w", err)
	}
	tempName := file.Name()
	defer os.Remove(tempName)
	if err := file.Chmod(0600); err != nil {
		_ = file.Close()
		return err
	}
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(config); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Link(tempName, h.ConfigFile); err != nil {
		return fmt.Errorf("lock installation config: %w", err)
	}
	if err := os.Remove(tempName); err != nil {
		return err
	}
	return nil
}

const installPage = `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>TG Group 安装向导</title>
<style>
:root{font-family:Inter,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;color:#172033;background:#f4f6fa}
*{box-sizing:border-box}body{margin:0;padding:36px 16px}.wrap{max-width:760px;margin:auto}.panel{background:#fff;border:1px solid #e5e9f0;border-radius:14px;padding:28px;box-shadow:0 8px 30px #18233a0a}
h1{font-size:24px;margin:0 0 8px}.sub{color:#64748b;margin:0 0 24px;line-height:1.6}h2{font-size:16px;margin:24px 0 14px}
.grid{display:grid;grid-template-columns:1fr 1fr;gap:14px}.field{display:flex;flex-direction:column;gap:7px}.field.full{grid-column:1/-1}label{font-size:13px;font-weight:600;color:#344054}
input,select{height:42px;width:100%;border:1px solid #d0d5dd;border-radius:8px;padding:0 11px;font:inherit;background:#fff}input:focus,select:focus{outline:2px solid #c7d7fe;border-color:#6172f3}
.actions{display:flex;gap:12px;margin-top:24px;flex-wrap:wrap}button{height:42px;border:0;border-radius:8px;padding:0 18px;font-weight:600;cursor:pointer;background:#3448c5;color:#fff}button.secondary{background:#eef2ff;color:#3448c5}button:disabled{opacity:.55;cursor:wait}
#result{margin-top:18px;padding:12px 14px;border-radius:8px;background:#f2f4f7;color:#344054;white-space:pre-wrap;line-height:1.6}#result.error{background:#fff1f0;color:#b42318}#result.success{background:#ecfdf3;color:#027a48}
.note{font-size:12px;color:#667085;line-height:1.6;margin-top:6px}@media(max-width:600px){.grid{grid-template-columns:1fr}.field.full{grid-column:auto}.panel{padding:20px}}
</style></head>
<body><main class="wrap"><section class="panel">
<h1>TG Group 安装向导</h1><p class="sub">配置 PostgreSQL 与 Redis，验证连接后初始化数据库并创建平台 Owner。密码只用于首次创建，不会写入安装配置文件。</p>
<form id="setup">
<h2>PostgreSQL 数据库</h2><div class="grid">
<div class="field"><label for="pgHost">主机地址</label><input id="pgHost" value="127.0.0.1" required></div>
<div class="field"><label for="pgPort">端口</label><input id="pgPort" type="number" value="5432" min="1" max="65535" required></div>
<div class="field"><label for="pgDB">数据库名</label><input id="pgDB" required></div>
<div class="field"><label for="pgUser">用户名</label><input id="pgUser" required></div>
<div class="field"><label for="pgPass">密码</label><input id="pgPass" type="password" autocomplete="new-password"></div>
<div class="field"><label for="pgSSL">SSL 模式</label><select id="pgSSL"><option>prefer</option><option>require</option><option>verify-ca</option><option>verify-full</option><option>disable</option></select></div>
</div>
<h2>Redis</h2><div class="grid">
<div class="field"><label for="redisAddr">地址（主机:端口）</label><input id="redisAddr" value="127.0.0.1:6379" required></div>
<div class="field"><label for="redisPass">密码（无密码可留空）</label><input id="redisPass" type="password" autocomplete="new-password"></div>
<div class="field"><label for="redisDB">数据库编号</label><input id="redisDB" type="number" min="0" value="0" required></div>
</div>
<h2>平台 Owner 账号</h2><div class="grid">
<div class="field"><label for="owner">管理员用户名</label><input id="owner" autocomplete="username" required maxlength="64"></div>
<div class="field"><label for="ownerPass">管理员密码（至少 12 位）</label><input id="ownerPass" type="password" autocomplete="new-password" minlength="12" maxlength="72" required></div>
</div><p class="note">请使用独立强密码。密码将以 bcrypt 哈希形式保存到 PostgreSQL，不会保存在安装配置文件中。</p>
<div class="actions"><button type="button" class="secondary" id="test">测试数据库连接</button><button type="submit" id="install">验证并完成安装</button></div>
</form><div id="result" role="status" aria-live="polite">等待配置。</div>
</section></main>
<script>
const $=id=>document.getElementById(id),result=$("result");
function payload(){return {postgresHost:$("pgHost").value,postgresPort:Number($("pgPort").value),postgresDatabase:$("pgDB").value,postgresUsername:$("pgUser").value,postgresPassword:$("pgPass").value,postgresSSLMode:$("pgSSL").value,redisAddress:$("redisAddr").value,redisPassword:$("redisPass").value,redisDB:Number($("redisDB").value),ownerUsername:$("owner").value,ownerPassword:$("ownerPass").value}}
async function request(path){const response=await fetch(path,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify(payload())});const body=await response.json();if(!response.ok||body.code!==0)throw new Error(body.message||"请求失败");return body.data}
$("test").addEventListener("click",async()=>{result.className="";result.textContent="正在测试 PostgreSQL 和 Redis…";$("test").disabled=true;try{await request("/api/install/test");result.className="success";result.textContent="PostgreSQL 与 Redis 连接成功。"}catch(e){result.className="error";result.textContent=e.message}finally{$("test").disabled=false}});
$("setup").addEventListener("submit",async e=>{e.preventDefault();$("install").disabled=true;result.className="";result.textContent="正在验证依赖、执行迁移并创建 Owner，请勿关闭页面…";try{const data=await request("/api/install/complete");result.className="success";result.textContent=data.message+"\n请在宝塔面板重启 TG Group 服务，然后重新访问网站。";$("setup").hidden=true}catch(e){result.className="error";result.textContent=e.message;$("install").disabled=false}});
</script></body></html>`
