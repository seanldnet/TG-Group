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
    "github.com/seanldnet/tg-group/internal/bots"
)

type Poller struct {
    token string
    auth *auth.Service
    bots *bots.Service
    logger *slog.Logger
    client *http.Client
}

type telegramUser struct {
    ID int64 `json:"id"`
    Username string `json:"username"`
    FirstName string `json:"first_name"`
    LastName string `json:"last_name"`
    IsBot bool `json:"is_bot"`
}
type telegramChat struct { ID int64 `json:"id"`; Type string `json:"type"` }
type telegramMessage struct {
    MessageID int64 `json:"message_id"`
    From *telegramUser `json:"from"`
    Chat telegramChat `json:"chat"`
    Text string `json:"text"`
}
type updateBatch struct {
    OK bool `json:"ok"`
    Result []struct {
        UpdateID int64 `json:"update_id"`
        Message *telegramMessage `json:"message"`
    } `json:"result"`
}

func New(token string, service *auth.Service, logger *slog.Logger) *Poller {
    return NewWithBotService(token,service,nil,logger)
}
func NewWithBotService(token string,service *auth.Service,botService *bots.Service,logger *slog.Logger)*Poller{
    if logger==nil {logger=slog.Default()}
    return &Poller{token:token,auth:service,bots:botService,logger:logger,client:&http.Client{Timeout:40*time.Second}}
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

func (p *Poller) handleMessage(ctx context.Context,message *telegramMessage) {
    if message == nil || message.From == nil || message.From.IsBot || message.Chat.Type != "private" || message.From.ID != message.Chat.ID { return }
    command := strings.TrimSpace(message.Text)
    if command=="/menu" || command=="主菜单" { _ = p.sendMessage(ctx,message.Chat.ID,"TG-Group 平台主菜单：请选择账号信息、添加群管 Bot 或查看使用帮助。"); return }
    if command=="添加群管 Bot" { _ = p.sendMessage(ctx,message.Chat.ID,"请发送命令：/addbot <Bot Token>。为保护凭据，平台会先删除这条私聊消息，再验证并加密保存 Token。"); return }
    if command=="使用帮助" || command=="/help" { _ = p.sendMessage(ctx,message.Chat.ID,"TG-Group 使用帮助：\n/start：注册账号或完成网站账号绑定\n/account：查询当前绑定账号\n/addbot <Bot Token>：添加群管 Bot\n网站控制台可管理群组和安全规则。"); return }
    if strings.HasPrefix(command,"/addbot") { p.addUserBot(ctx,message); return }
    if command == "/account" || command == "账号信息" {
        account, err := p.auth.TelegramAccount(ctx,message.From.ID)
        if err != nil {
            _ = p.sendMessage(ctx,message.Chat.ID,"当前 Telegram 尚未绑定平台账号。发送 /start 可以注册新账号，或先登录网站后在账号安全中发起绑定。")
            return
        }
        text := "TG-Group 账号信息\\n用户 ID："+account.UserID+"\\n登录账号："+account.UserID+"\\n邮箱："+account.Email+"\\n昵称："+account.DisplayName+"\\n角色："+account.Role
        if err := p.sendMessage(ctx,message.Chat.ID,text); err != nil { p.logger.Warn("send Telegram account summary failed","error",err) }
        return
    }
    parts := strings.Fields(message.Text)
    if len(parts) == 0 || parts[0] != "/start" { return }
    identity := auth.TelegramIdentity{
        ID:message.From.ID, Username:message.From.Username, FirstName:message.From.FirstName, LastName:message.From.LastName,
    }
    if len(parts) == 1 {
        registration, err := p.auth.RegisterTelegramAccount(ctx, identity)
        if err != nil {
            if err == auth.ErrTelegramAlreadyLinked {
                _ = p.sendMessage(ctx,message.Chat.ID,"这个 Telegram 账号已经注册或绑定过 TG-Group。请使用原账号登录网站，或在账号安全中发起绑定。")
                return
            }
            p.logger.Error("Telegram platform registration failed","telegram_user_id",message.From.ID,"error",err)
            _ = p.sendMessage(ctx,message.Chat.ID,"注册暂时无法完成，请稍后重试或联系平台管理员。")
            return
        }
        text := "TG-Group 账号注册成功！\n用户 ID："+registration.UserID+"\n登录账号："+registration.UserID+"\n初始密码："+registration.Password+"\n请立即登录网站并修改初始密码。"
        if err := p.sendMessage(ctx,message.Chat.ID,text); err != nil {
            p.logger.Warn("send Telegram registration credentials failed","error",err)
        }
        return
    }
    if len(parts) != 2 { return }
    payload := strings.TrimSpace(parts[1])
    if payload == "" || len(payload) > 128 { return }
    err := p.auth.CompleteTelegramLink(ctx,payload,identity)
    if err != nil {
        p.logger.Info("Telegram account binding was not completed","telegram_user_id",message.From.ID,"reason",safeReason(err))
        if sendErr := p.sendMessage(ctx,message.Chat.ID,"绑定未完成：绑定链接可能已过期，或该 Telegram 账号/平台账号已绑定。请返回网站重新发起绑定。"); sendErr != nil {
            p.logger.Warn("send Telegram binding error failed","error",sendErr)
        }
        return
    }
    if err := p.sendMessage(ctx,message.Chat.ID,"Telegram 账号绑定成功。你可以返回 TG-Group 网站查看绑定状态。"); err != nil {
        p.logger.Warn("send Telegram binding confirmation failed","error",err)
    }
}

func (p *Poller) addUserBot(ctx context.Context,message *telegramMessage) {
    parts:=strings.Fields(message.Text)
    if len(parts)!=2 || (parts[0]!="/addbot" && !strings.HasPrefix(parts[0],"/addbot@")) {
        _ = p.sendMessage(ctx,message.Chat.ID,"用法：/addbot <Bot Token>。请从 BotFather 复制完整 Token。")
        return
    }
    if p.bots==nil {
        _ = p.sendMessage(ctx,message.Chat.ID,"当前平台暂未启用群管 Bot Token 管理服务，请联系管理员。")
        return
    }
    account,err:=p.auth.TelegramAccount(ctx,message.From.ID)
    if err!=nil {
        _ = p.sendMessage(ctx,message.Chat.ID,"请先使用 /start 注册平台账号，或在网站账号安全中绑定 Telegram 后再添加群管 Bot。")
        return
    }
    token:=parts[1]
    if err:=p.deleteMessage(ctx,message.Chat.ID,message.MessageID);err!=nil {
        p.logger.Warn("refuse Telegram Bot token because source message could not be deleted","telegram_user_id",message.From.ID,"error",err)
        _ = p.sendMessage(ctx,message.Chat.ID,"为保护 Token，平台未能删除你刚发送的消息，因此没有处理该 Token。请立即手动删除这条消息，然后改用网站机器人管理添加。")
        return
    }
    bot,err:=p.bots.Add(ctx,account.TenantID,account.UserID,token)
    if err!=nil {
        switch {
        case errors.Is(err,bots.ErrTokenTaken): _ = p.sendMessage(ctx,message.Chat.ID,"这个群管 Bot 已经被平台接入，请检查是否重复添加。")
        case errors.Is(err,bots.ErrInvalidToken): _ = p.sendMessage(ctx,message.Chat.ID,"Telegram 不接受该 Bot Token，请检查后重新添加。")
        case errors.Is(err,bots.ErrTelegramUnavailable): _ = p.sendMessage(ctx,message.Chat.ID,"Telegram 验证服务暂时不可用，请稍后重试。")
        default: p.logger.Error("add user Telegram Bot from platform Bot","telegram_user_id",message.From.ID,"error",err);_ = p.sendMessage(ctx,message.Chat.ID,"添加 Bot 失败，请稍后重试或联系管理员。")
        }
        return
    }
    label:=bot.FirstName
    if bot.Username!="" {label="@"+bot.Username}
    _ = p.sendMessage(ctx,message.Chat.ID,"群管 Bot 已验证并添加成功："+label+"。你可以将它添加到群组并授予所需管理权限。")
}

func (p *Poller) deleteMessage(ctx context.Context,chatID,messageID int64)error{
    payload,_:=json.Marshal(map[string]any{"chat_id":chatID,"message_id":messageID})
    req,err:=http.NewRequestWithContext(ctx,http.MethodPost,"https://api.telegram.org/bot"+p.token+"/deleteMessage",strings.NewReader(string(payload)))
    if err!=nil{return err}
    req.Header.Set("Content-Type","application/json")
    response,err:=p.client.Do(req);if err!=nil{return err};defer response.Body.Close()
    data,err:=io.ReadAll(io.LimitReader(response.Body,1<<20));if err!=nil{return err}
    if response.StatusCode!=http.StatusOK{return fmt.Errorf("Telegram deleteMessage returned HTTP %d",response.StatusCode)}
    var result struct{OK bool `json:"ok"`}
    if json.Unmarshal(data,&result)!=nil||!result.OK{return fmt.Errorf("Telegram deleteMessage returned ok=false")}
    return nil
}

func (p *Poller) sendMessage(ctx context.Context, chatID int64, text string) error {
    payload,_ := json.Marshal(map[string]any{"chat_id":chatID,"text":text,"reply_markup":map[string]any{"keyboard":[][]string{{"账号信息"},{"添加群管 Bot","使用帮助"}},"resize_keyboard":true,"is_persistent":true}})
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
