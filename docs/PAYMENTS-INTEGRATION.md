# EPUSDT / GMPay 在线支付集成核对

> 核对日期：2026-11-27。此文档记录已核对的官方公开商户协议，不代表支付渠道已接入生产。

## 已核对的官方协议

官方文档：
- EPUSDT Merchant API：https://www.epusdt.com/api/payment
- EPUSDT API Reference：https://www.epusdt.com/api/reference
- EPUSDT 中文 Merchant API：https://www.epusdt.com/zh/api/payment

### GMPay 创建订单

- `POST /payments/gmpay/v1/order/create-transaction`
- JSON 或 `application/x-www-form-urlencoded`。
- 必需字段：`pid`、`order_id`（最多 32 字符）、`currency`、`amount`、`notify_url`、`signature`。
- 选择链上资产时，`token` 和 `network` 必须同时提供；二者同时省略则创建等待用户选择的 placeholder。
- 应先读取 `GET /payments/gmpay/v1/config` 的 `data.supported_assets`，不能硬编码或假设 TRON/USDT 一定可用。
- 成功响应包含 `trade_id`、`order_id`、`actual_amount`、`token`、`status`、`expiration_time`、`payment_url`。

### GMPay 请求与回调签名

- GMPay 使用 HMAC-SHA256；排除 `signature`、空值和 null，按字段名 ASCII 升序排序，以未转义的 `key=value` 用 `&` 连接，使用商户 secret_key 作为 HMAC key，输出小写 64 位十六进制。
- GMPay 回调为 JSON POST；按 `pid` 查找本地商户密钥，用相同规则验证。回调示例字段含 `trade_id`、`order_id`、`amount`、`actual_amount`、`token`、`block_transaction_id`、`status=2`。回调体不包含 `payment_type`。
- 官方要求先验签、匹配订单和金额、重复通知幂等处理；成功处理后返回 HTTP 200 纯文本 `ok` 或 `success`。
- 订单状态：1 待支付、2 已支付、3 已过期、4 等待选择、5 已观察到转账并确认中、6 严格验证失败并进入人工复核。状态 5 不能提前入账；只有最终支付成功状态才可完成本地业务。

### EPay 兼容协议

- `GET/POST /payments/epay/v1/order/create-transaction/submit.php`。
- EPay 使用 MD5 兼容签名：排除 `sign` 和 `sign_type`，按字段名排序，拼接非空 `key=value` 后追加 secret，再计算小写 MD5。
- 回调为 GET，成功交易状态为 `TRADE_SUCCESS`；必须核验签名、商户 PID、商户订单号、金额和交易状态。
- GMPay 为新集成优先协议；EPay 仅作为兼容路径，不能将两者的签名算法混用。

## 本项目实施约束

1. 钱包金额及代理等级价格继续使用 PostgreSQL BIGINT；禁止 Go float32/float64 参与入账或比较。
2. 官方响应的 `actual_amount` 是小数形式的加密资产数量，而本项目需求规定 USDT 金额使用整数。实现前必须明确整数口径：若坚持整 USDT，则拒绝非整数实际到账金额；不得静默四舍五入或截断。若产品实际要求最小精度单位，应先调整需求与数据库金额单位定义。
3. 订单创建前先在本地事务创建 pending 充值订单及唯一商户订单号，再请求网关；保存网关 `trade_id`、checkout URL、币种、网络、应付数量、过期时间。外部请求超时需通过本地订单号主动查单/重试，不可重复生成多笔可收款订单。
4. 回调必须使用原始/规范化字段做 HMAC 验签；随后在一个数据库事务中锁定本地订单、核对 PID、merchant order ID、trade ID、币种、网络、金额和唯一链上交易引用，幂等更新订单、钱包余额和 wallet_ledger。
5. 钱包入账使用唯一幂等键和 ledger 唯一约束；回调只有在事务提交成功后才返回 `ok`。重复回调返回成功响应但不得二次入账。
6. 浏览器跳转或用户端状态轮询不能作为支付入账凭证；服务端回调验签与服务端主动查单是可信来源。
7. 网关 Base URL、PID、Secret、currency、network/token、notify URL 必须由管理员安全配置；secret 不得返回前端、写日志或提交到仓库。
8. 生产接入仍需取得实际 EPUSDT 网关 Base URL、PID/Secret、可用网络/Token 列表、沙箱/测试环境和公网 HTTPS notify URL 后，才能完成联调验收。

## 当前代码状态

- 已新增 GMPay HMAC-SHA256 签名和常量时间验签原语，并以官方文档公开示例向量编写测试。
- 尚未实现网关配置、创建充值订单、查单、GMPay/EPay 回调路由、钱包入账和前端充值流程；不得宣称在线支付已上线。
