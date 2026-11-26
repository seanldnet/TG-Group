package telegramauthbot

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "html"
    "io"
    "log/slog"
    "net/http"
    "net/url"
    "strconv"
    "strings"
    "sync"
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
    stateMu sync.Mutex
    awaitingBotToken map[int64]time.Time
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
type telegramCallbackQuery struct {
    ID string `json:"id"`
    From *telegramUser `json:"from"`
    Message *telegramMessage `json:"message"`
    Data string `json:"data"`
}
type updateBatch struct {
    OK bool `json:"ok"`
    Result []struct {
        UpdateID int64 `json:"update_id"`
        Message *telegramMessage `json:"message"`
        CallbackQuery *telegramCallbackQuery `json:"callback_query"`
    } `json:"result"`
}

func New(token string, service *auth.Service, logger *slog.Logger) *Poller {
    return NewWithBotService(token,service,nil,logger)
}
func NewWithBotService(token string,service *auth.Service,botService *bots.Service,logger *slog.Logger)*Poller{
    if logger==nil {logger=slog.Default()}
    return &Poller{token:token,auth:service,bots:botService,logger:logger,client:&http.Client{Timeout:40*time.Second},awaitingBotToken:make(map[int64]time.Time)}
}

func (p *Poller) Run(ctx context.Context) {
    var offset int64
    for ctx.Err() == nil {
        query := url.Values{"timeout":{"25"},"allowed_updates":{ "[\"message\",\"callback_query\"]" }}
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
            if item.CallbackQuery != nil { p.handleCallback(ctx,item.CallbackQuery) }; p.handleMessage(ctx,item.Message)
        }
    }
}

func (p *Poller) handleMessage(ctx context.Context,message *telegramMessage) {
    if message == nil || message.From == nil || message.From.IsBot || message.Chat.Type != "private" || message.From.ID != message.Chat.ID { return }
    if p.consumeBotTokenPrompt(message.From.ID) {
        p.addUserBot(ctx,message,strings.TrimSpace(message.Text))
        return
    }
    text := strings.TrimSpace(message.Text)
    parts := strings.Fields(text)
    if len(parts)>0 && parts[0]=="/start" {
        identity := auth.TelegramIdentity{ID:message.From.ID,Username:message.From.Username,FirstName:message.From.FirstName,LastName:message.From.LastName}
        if len(parts)==2 {
            payload:=strings.TrimSpace(parts[1])
            if payload!="" && len(payload)<=128 {
                if err:=p.auth.CompleteTelegramLink(ctx,payload,identity);err!=nil {
                    p.logger.Info("Telegram account binding was not completed","telegram_user_id",message.From.ID,"reason",safeReason(err))
                    _=p.sendMessage(ctx,message.Chat.ID,"绑定未完成：链接可能已过期，或账号已绑定。请返回网站重新发起绑定。")
                } else { _=p.sendMessage(ctx,message.Chat.ID,"Telegram 账号绑定成功。") }
                return
            }
        }
        account,lookupErr:=p.auth.TelegramAccount(ctx,message.From.ID)
        if lookupErr==nil {
            _=p.sendMessage(ctx,message.Chat.ID,"欢迎回来，"+account.DisplayName+"。请选择下方菜单操作。")
            return
        }
        _=p.sendIntroMenu(ctx,message.Chat.ID)
        return
    }
    replyActions:=map[string]string{
        "平台介绍":"menu:intro","开始注册":"menu:register","账号信息":"menu:account",
        "添加群管 Bot":"menu:addbot","使用帮助":"menu:help","取消操作":"menu:cancel",
    }
    if action,ok:=replyActions[text];ok {
        p.handleCallback(ctx,&telegramCallbackQuery{From:message.From,Message:message,Data:action})
        return
    }
    // All day-to-day actions are button-driven. Ignore legacy slash commands.
}
func (p *Poller) consumeBotTokenPrompt(userID int64) bool {
    p.stateMu.Lock(); defer p.stateMu.Unlock()
    deadline,ok:=p.awaitingBotToken[userID]
    delete(p.awaitingBotToken,userID)
    return ok && time.Now().Before(deadline)
}
func (p *Poller) handleCallback(ctx context.Context,query *telegramCallbackQuery) {
    if query==nil || query.From==nil || query.From.IsBot || query.Message==nil || query.Message.Chat.Type!="private" || query.Message.Chat.ID!=query.From.ID { return }
    if query.ID!="" { _=p.answerCallback(ctx,query.ID) }
    switch query.Data {
    case "menu:intro":
        _=p.sendIntroMenu(ctx,query.Message.Chat.ID)
    case "menu:register":
        identity:=auth.TelegramIdentity{ID:query.From.ID,Username:query.From.Username,FirstName:query.From.FirstName,LastName:query.From.LastName}
        if _,err:=p.auth.TelegramAccount(ctx,query.From.ID);err==nil {
            _=p.sendMessage(ctx,query.Message.Chat.ID,"这个 Telegram 账号已经有平台账号，请使用“账号信息”查看资料。")
            return
        }
        registration,err:=p.auth.RegisterTelegramAccount(ctx,identity)
        if err!=nil {
            if errors.Is(err,auth.ErrTelegramAlreadyLinked) {
                _=p.sendMessage(ctx,query.Message.Chat.ID,"这个 Telegram 账号已经注册，请使用“账号信息”查看资料。")
                return
            }
            p.logger.Error("Telegram platform registration failed","telegram_user_id",query.From.ID,"error",err)
            _=p.sendMessage(ctx,query.Message.Chat.ID,"注册暂时无法完成，请稍后重试或联系平台管理员。")
            return
        }
        _=p.sendRegistrationCredentials(ctx,query.Message.Chat.ID,registration)
    case "menu:account":
        account,err:=p.auth.TelegramAccount(ctx,query.From.ID)
        if err!=nil { _=p.sendMessage(ctx,query.Message.Chat.ID,"当前 Telegram 尚未关联平台账号。请先在网站“账号设置 → 账号关联”完成绑定，或使用开始按钮注册。");return }
        _=p.sendMessage(ctx,query.Message.Chat.ID,"账号信息\n用户 ID："+account.UserID+"\n邮箱："+account.Email+"\n昵称："+account.DisplayName+"\n角色："+account.Role)
    case "menu:addbot":
        if _,err:=p.auth.TelegramAccount(ctx,query.From.ID);err!=nil { _=p.sendMessage(ctx,query.Message.Chat.ID,"请先完成平台账号注册或 Telegram 账号关联，再添加群管 Bot。");return }
        if p.bots==nil { _=p.sendMessage(ctx,query.Message.Chat.ID,"群管 Bot 管理服务暂不可用，请联系管理员。");return }
        p.stateMu.Lock();p.awaitingBotToken[query.From.ID]=time.Now().Add(5*time.Minute);p.stateMu.Unlock()
        _=p.sendMessage(ctx,query.Message.Chat.ID,"请直接发送要添加的 Bot Token。收到后会先尝试删除这条消息，再验证并加密保存。若无法删除，系统将拒绝处理。")
    case "menu:help":
        _=p.sendMessage(ctx,query.Message.Chat.ID,"使用帮助\n• 账号信息：查看当前绑定的平台账号\n• 添加群管 Bot：点击按钮后直接发送 BotFather 提供的 Token\n• 网站控制台：管理群组、成员、规则和安全设置\n所有日常操作请使用下方按钮，无需输入命令。")
    case "menu:cancel":
        p.stateMu.Lock();delete(p.awaitingBotToken,query.From.ID);p.stateMu.Unlock()
        _=p.sendMessage(ctx,query.Message.Chat.ID,"已取消当前操作。")
    }
}
func (p *Poller) addUserBot(ctx context.Context,message *telegramMessage,token string) {
    if p.bots==nil { _=p.sendMessage(ctx,message.Chat.ID,"群管 Bot 管理服务暂不可用，请联系管理员。");return }
    account,err:=p.auth.TelegramAccount(ctx,message.From.ID)
    if err!=nil { _=p.sendMessage(ctx,message.Chat.ID,"请先完成平台账号注册或 Telegram 账号关联。");return }
    if err:=p.deleteMessage(ctx,message.Chat.ID,message.MessageID);err!=nil {
        p.logger.Warn("refuse Telegram Bot token because source message could not be deleted","telegram_user_id",message.From.ID,"error",err)
        _=p.sendMessage(ctx,message.Chat.ID,"出于安全考虑，系统未能删除你发送的 Token 消息，因此没有处理它。请立即手动删除该消息，并改用网站机器人管理添加。")
        return
    }
    bot,err:=p.bots.Add(ctx,account.TenantID,account.UserID,token)
    if err!=nil {
        switch {
        case errors.Is(err,bots.ErrTokenTaken): _=p.sendMessage(ctx,message.Chat.ID,"这个群管 Bot 已经接入平台，请勿重复添加。")
        case errors.Is(err,bots.ErrInvalidToken): _=p.sendMessage(ctx,message.Chat.ID,"Telegram 不接受该 Bot Token，请检查后重新操作。")
        case errors.Is(err,bots.ErrTelegramUnavailable): _=p.sendMessage(ctx,message.Chat.ID,"Telegram 验证服务暂时不可用，请稍后重试。")
        default: p.logger.Error("add user Telegram Bot from platform Bot","telegram_user_id",message.From.ID,"error",err);_=p.sendMessage(ctx,message.Chat.ID,"添加 Bot 失败，请稍后重试或联系平台管理员。")
        }
        return
    }
    label:=bot.FirstName;if bot.Username!="" {label="@"+bot.Username}
    _=p.sendMessage(ctx,message.Chat.ID,"群管 Bot 已成功添加："+label+"。现在可以将它加入目标群组并授予管理权限。")
}
func (p *Poller) answerCallback(ctx context.Context,callbackID string) error {
    payload,_:=json.Marshal(map[string]string{"callback_query_id":callbackID})
    req,err:=http.NewRequestWithContext(ctx,http.MethodPost,"https://api.telegram.org/bot"+p.token+"/answerCallbackQuery",strings.NewReader(string(payload)))
    if err!=nil{return err};req.Header.Set("Content-Type","application/json")
    response,err:=p.client.Do(req);if err!=nil{return err};defer response.Body.Close()
    _,_=io.Copy(io.Discard,io.LimitReader(response.Body,1<<20))
    if response.StatusCode!=http.StatusOK{return fmt.Errorf("Telegram answerCallbackQuery returned HTTP %d",response.StatusCode)}
    return nil
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

func (p *Poller) sendMessage(ctx context.Context, chatID int64, text string) error { return p.sendMessageWithParseMode(ctx,chatID,text,"") }

func registrationCredentialsHTML(registration auth.TelegramRegistration) string {
    return "账号注册成功！\n用户 ID：<code>"+html.EscapeString(registration.UserID)+"</code>\n初始密码：<code>"+html.EscapeString(registration.Password)+"</code>\n\n点击上方每个代码框即可复制。请登录网站后立即修改初始密码。"
}

func (p *Poller) sendRegistrationCredentials(ctx context.Context,chatID int64,registration auth.TelegramRegistration) error {
    return p.sendMessageWithParseMode(ctx,chatID,registrationCredentialsHTML(registration),"HTML")
}

func platformIntroText() string {
    return "欢迎使用 TG-Group 群组管理平台！\n\n" +
        "TG-Group 是一个多机器人、多群组的 Telegram 管理控制台。你可以把多个群管 Bot 接入同一个平台账号，在一个网站后台集中管理，不必为每个 Bot 重复设置主人。\n\n" +
        "接入 Bot 并加入群组、授予必要的管理员权限后，可在平台逐步配置和查看：\n" +
        "• 群组与机器人：自动发现群组、查看 Bot 在线状态和管理权限\n" +
        "• 消息与成员：查看群消息记录、成员资料和成员状态\n" +
        "• 自动化群规：关键词/敏感词过滤、反广告、反刷屏、欢迎语和退群通知\n" +
        "• 入群验证：数学题与频道关注验证（按群组配置）\n" +
        "• 成员管理：查看权限并执行封禁、解封、禁言和解除禁言\n\n" +
        "开始使用：先注册或关联平台账号，再添加群管 Bot；然后把 Bot 拉入群组并授予所需权限，最后在网站控制台配置群规。所有 Bot 都会使用账号关联中的 Telegram 主人身份。\n\n" +
        "已有平台账号请点“账号信息”。首次使用请点“开始注册”。"
}

func (p *Poller) sendIntroMenu(ctx context.Context,chatID int64) error {
    return p.sendMessageWithKeyboard(ctx,chatID,platformIntroText(),"",platformIntroKeyboard())
}

func platformIntroKeyboard() map[string]any {
    return map[string]any{"inline_keyboard":[][]map[string]string{
        {{"text":"开始注册","callback_data":"menu:register"},{"text":"账号信息","callback_data":"menu:account"}},
        {{"text":"添加群管 Bot","callback_data":"menu:addbot"},{"text":"使用帮助","callback_data":"menu:help"}},
    }}
}

func platformReplyKeyboard() map[string]any {
    return map[string]any{
        "keyboard":[][]map[string]string{
            {{"text":"平台介绍"},{"text":"账号信息"}},
            {{"text":"添加群管 Bot"},{"text":"使用帮助"}},
            {{"text":"取消操作"}},
        },
        "resize_keyboard":true,
        "is_persistent":true,
        "input_field_placeholder":"请选择菜单操作…",
    }
}

func (p *Poller) sendMessageWithParseMode(ctx context.Context, chatID int64, text string, parseMode string) error {
    keyboard:=map[string]any{"inline_keyboard":[][]map[string]string{
        {{"text":"账号信息","callback_data":"menu:account"},{"text":"添加群管 Bot","callback_data":"menu:addbot"}},
        {{"text":"使用帮助","callback_data":"menu:help"},{"text":"取消当前操作","callback_data":"menu:cancel"}},
    }}
    return p.sendMessageWithKeyboard(ctx,chatID,text,parseMode,keyboard)
}

func (p *Poller) sendMessageWithKeyboard(ctx context.Context,chatID int64,text,parseMode string,keyboard map[string]any) error {
    payloadData:=map[string]any{"chat_id":chatID,"text":text,"reply_markup":map[string]any{"inline_keyboard":keyboard["inline_keyboard"],"keyboard":platformReplyKeyboard()["keyboard"],"resize_keyboard":true,"is_persistent":true,"input_field_placeholder":"请选择菜单操作…"}}
    if parseMode!="" { payloadData["parse_mode"]=parseMode }
    payload,_:=json.Marshal(payloadData)
    req,err:=http.NewRequestWithContext(ctx,http.MethodPost,"https://api.telegram.org/bot"+p.token+"/sendMessage",strings.NewReader(string(payload)))
    if err!=nil{return err};req.Header.Set("Content-Type","application/json")
    response,err:=p.client.Do(req);if err!=nil{return err};defer response.Body.Close()
    data,err:=io.ReadAll(io.LimitReader(response.Body,1<<20));if err!=nil{return err}
    if response.StatusCode!=http.StatusOK{return fmt.Errorf("Telegram sendMessage returned HTTP %d",response.StatusCode)}
    var result struct{OK bool `json:"ok"`}
    if json.Unmarshal(data,&result)!=nil||!result.OK{return fmt.Errorf("Telegram sendMessage returned ok=false")}
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
