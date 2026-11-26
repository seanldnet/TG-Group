package telegramauthbot

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "log/slog"
    "net/http"
    "net/url"
    "strconv"
    "strings"
    "time"

    "github.com/seanldnet/tg-group/internal/auth"
)

type Poller struct {
    token string
    auth *auth.Service
    logger *slog.Logger
    client *http.Client
}

type updateBatch struct {
    OK bool `json:"ok"`
    Result []struct {
        UpdateID int64 `json:"update_id"`
        Message *struct {
            From *struct {
                ID int64 `json:"id"`
                Username string `json:"username"`
                FirstName string `json:"first_name"`
                LastName string `json:"last_name"`
                IsBot bool `json:"is_bot"`
            } `json:"from"`
            Chat struct { ID int64 `json:"id"`; Type string `json:"type"` } `json:"chat"`
            Text string `json:"text"`
        } `json:"message"`
    } `json:"result"`
}

func New(token string, service *auth.Service, logger *slog.Logger) *Poller {
    if logger == nil { logger = slog.Default() }
    return &Poller{token: token, auth: service, logger: logger, client: &http.Client{Timeout: 40*time.Second}}
}

func (p *Poller) Run(ctx context.Context) {
    var offset int64
    for ctx.Err() == nil {
        query := url.Values{"timeout":{"25"},"allowed_updates":{ "[\"message\"]" }}
        if offset > 0 { query.Set("offset", strconv.FormatInt(offset,10)) }
        req, err := http.NewRequestWithContext(ctx,http.MethodGet,"https://api.telegram.org/bot"+p.token+"/getUpdates?"+query.Encode(),nil)
        if err != nil { p.logger.Error("create Telegram auth Bot request failed"); return }
        response, err := p.client.Do(req)
        if err != nil {
            if ctx.Err() != nil { return }
            p.logger.Warn("Telegram auth Bot polling request failed")
            wait(ctx,3*time.Second); continue
        }
        body, readErr := io.ReadAll(io.LimitReader(response.Body,2<<20))
        response.Body.Close()
        if readErr != nil || response.StatusCode != http.StatusOK {
            p.logger.Warn("Telegram auth Bot returned invalid polling response","status",response.StatusCode)
            wait(ctx,3*time.Second); continue
        }
        var batch updateBatch
        if err := json.Unmarshal(body,&batch); err != nil || !batch.OK {
            p.logger.Warn("Telegram auth Bot update payload could not be decoded")
            wait(ctx,3*time.Second); continue
        }
        for _, item := range batch.Result {
            if item.UpdateID >= offset { offset = item.UpdateID+1 }
            p.handleMessage(ctx,item.Message)
        }
    }
}

func (p *Poller) handleMessage(ctx context.Context, message *struct {
    From *struct { ID int64 `json:"id"`; Username string `json:"username"`; FirstName string `json:"first_name"`; LastName string `json:"last_name"`; IsBot bool `json:"is_bot"` } `json:"from"`
    Chat struct { ID int64 `json:"id"`; Type string `json:"type"` } `json:"chat"`
    Text string `json:"text"`
}) {
    if message == nil || message.From == nil || message.From.IsBot || message.Chat.Type != "private" || message.From.ID != message.Chat.ID { return }
    parts := strings.Fields(message.Text)
    if len(parts) != 2 || parts[0] != "/start" { return }
    payload := strings.TrimSpace(parts[1])
    if payload == "" || len(payload) > 128 { return }
    err := p.auth.CompleteTelegramLink(ctx,payload,auth.TelegramIdentity{
        ID:message.From.ID, Username:message.From.Username, FirstName:message.From.FirstName, LastName:message.From.LastName,
    })
    if err != nil {
        p.logger.Info("Telegram account binding was not completed","telegram_user_id",message.From.ID,"reason",safeReason(err))
        return
    }
    if err := p.sendMessage(ctx,message.Chat.ID,"Telegram 账号绑定成功。你可以返回 TG-Group 网站查看绑定状态。"); err != nil {
        p.logger.Warn("send Telegram binding confirmation failed","error",err)
    }
}

func (p *Poller) sendMessage(ctx context.Context, chatID int64, text string) error {
    payload,_ := json.Marshal(map[string]any{"chat_id":chatID,"text":text})
    req,err := http.NewRequestWithContext(ctx,http.MethodPost,"https://api.telegram.org/bot"+p.token+"/sendMessage",strings.NewReader(string(payload)))
    if err != nil { return err }
    req.Header.Set("Content-Type","application/json")
    response,err := p.client.Do(req)
    if err != nil { return err }
    defer response.Body.Close()
    data,err := io.ReadAll(io.LimitReader(response.Body,1<<20))
    if err != nil { return err }
    if response.StatusCode != http.StatusOK { return fmt.Errorf("Telegram sendMessage returned HTTP %d",response.StatusCode) }
    var result struct{ OK bool `json:"ok"` }
    if err := json.Unmarshal(data,&result); err != nil { return err }
    if !result.OK { return fmt.Errorf("Telegram sendMessage returned ok=false") }
    return nil
}

func safeReason(err error) string {
    switch err {
    case auth.ErrTelegramAlreadyLinked: return "telegram_identity_already_linked"
    case auth.ErrTelegramAlreadyBound: return "user_already_has_telegram_identity"
    case auth.ErrTelegramChallengeExpired: return "challenge_invalid_or_expired"
    default: return "binding_failed"
    }
}
func wait(ctx context.Context, duration time.Duration) {
    timer:=time.NewTimer(duration); defer timer.Stop()
    select { case <-ctx.Done(): case <-timer.C: }
}
