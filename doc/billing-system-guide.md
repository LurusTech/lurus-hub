# Lurus 计费系统说明

## 目录
- [系统架构](#系统架构)
- [支持的支付方式](#支持的支付方式)
- [计费模式](#计费模式)
- [收费流程](#收费流程)
- [多产品计费能力评估](#多产品计费能力评估)
- [配置指南](#配置指南)
- [常见问题](#常见问题)

---

## 系统架构

### 核心概念

```
┌─────────────────────────────────────────────────┐
│           Lurus 计费系统                          │
├─────────────────────────────────────────────────┤
│                                                 │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐     │
│  │ 充值系统 │  │ 订阅系统 │  │ 扣费系统 │     │
│  └──────────┘  └──────────┘  └──────────┘     │
│        │              │              │         │
│        └──────────────┴──────────────┘         │
│                      │                         │
│              ┌───────▼────────┐                │
│              │   额度管理      │                │
│              └────────────────┘                │
│                                                 │
└─────────────────────────────────────────────────┘
                     │
          ┌──────────┴──────────┐
          │                     │
    ┌─────▼─────┐        ┌─────▼─────┐
    │  产品 A    │        │  产品 B    │
    │ (tenant_a) │        │ (tenant_b) │
    └───────────┘        └───────────┘
```

### 关键组件

| 组件 | 职责 | 数据表 |
|------|------|--------|
| **充值系统** | 一次性购买额度 | `topups` |
| **订阅系统** | 周期性订阅套餐 | `subscriptions`, `subscription_plans` |
| **额度管理** | 追踪用户剩余额度、日限额 | `users.quota`, `users.daily_quota` |
| **扣费系统** | AI 调用后自动扣费 | `logs` |
| **兑换码** | 批量发放额度 | `redemptions` |

---

## 支持的支付方式

### 已集成支付渠道

| 支付方式 | 代码标识 | 适用地区 | 手续费 | 状态 | 说明 |
|---------|---------|---------|--------|------|------|
| **Stripe** | `stripe` | 🌍 全球 | 2.9% + $0.30 | ✅ 生产可用 | Checkout Session + Webhook 验签 |
| **易支付 (Epay)** | `epay` | 🇨🇳 中国 | 自定义 | ✅ 生产可用 | 聚合支付商，可走支付宝/微信/银联通道 |
| **Creem** | `creem` | 🌍 全球 | 3.5% | ✅ 生产可用 | 固定产品定价，Webhook 验签 |
| **支付宝** | `alipay` | 🇨🇳 中国 | 0.6% | ❌ 仅 OAuth 登录 | 无支付层，直连需企业资质 |
| **微信支付** | `wechat` | 🇨🇳 中国 | 0.6% | ❌ 仅 OAuth 登录 | 无支付层，直连需企业资质 |

> **国内支付说明**：通过**易支付**已可走支付宝和微信通道（中间商方式，约 1-2% 手续费）。
> 若需官方直连，须持有企业营业执照并通过平台审核。

### 配置方式

**通过系统设置 → 支付配置**

```bash
# 1. Stripe
STRIPE_SECRET_KEY=sk_live_xxxxx
STRIPE_WEBHOOK_SECRET=whsec_xxxxx

# 2. Epay (易支付，含支付宝/微信通道)
EPAY_URL=https://pay.example.com
EPAY_PID=12345
EPAY_KEY=xxxxx

# 3. Creem
CREEM_API_KEY=xxxxx
CREEM_SECRET=xxxxx
```

---

## 计费模式

### 模式 1：预付费（充值）

**特点：**
- ✅ 用户先充值，后使用
- ✅ 按实际消耗扣费
- ✅ 灵活，适合低频用户

**充值流程：**
```
用户 → 选择充值金额 (¥100) → 支付 → 到账额度 (5000万 quota 单位)
```

**换算关系（可配置）：**
```
QuotaPerUnit = 500,000 (每元对应 quota 单位，可按汇率配置)
¥1 = 500,000 quota 单位
¥100 = 50,000,000 quota 单位
```

**API 端点：**
```bash
# 1. 获取充值信息
GET /api/user/topup

# 2. 发起易支付
POST /api/user/pay
{ "amount": 100, "payment_method": "alipay" }

# 3. 回调通知
GET /api/user/epay/notify  (易支付异步通知)
POST /api/pay/stripe (Stripe Webhook)
POST /api/pay/creem  (Creem Webhook)
```

---

### 模式 2：订阅制

**特点：**
- ✅ 周期性额度，包含日限额
- ✅ 到期自动从余额扣费续费（`auto_renew=true`）
- ✅ 适合高频用户

**默认套餐：**

| 套餐 | 价格 | 总额度 | 日限额 | 有效期 |
|------|------|--------|--------|--------|
| 周付 | ¥19.9 | 500万 quota | 50万/天 | 7天 |
| 月付 | ¥59.9 | 5000万 quota | 100万/天 | 30天 |
| 季付 | ¥149.9 | 2亿 quota | 200万/天 | 90天 |
| 年付 | ¥499.9 | 无上限 | 500万/天 | 365天 |

**订阅流程：**
```
用户 → 选择套餐 → 支付 → 订阅生效 → 到期前 24h 尝试自动扣费续费
```

**自动续费逻辑（已实现）：**
```
1. Cron 每小时检查：expires_at 在 24h 内 + auto_renew=true
2. 计算续费成本 = plan.Price * QuotaPerUnit
3. 检查 user.Quota >= 续费成本
   - 足够：原子事务扣费 + 延长 expires_at + 补充新期 TotalQuota
   - 不足：记录日志（待接入邮件通知）
4. 写入充值日志
```

---

### 模式 3：混合模式（推荐）

**特点：**
- ✅ 订阅 + 充值并存
- ✅ 订阅提供日限额上限，充值增加余额
- ✅ 适合所有场景

**计算逻辑：**
```
1. 用户 quota 是统一余额池（订阅激活时加 TotalQuota，充值时加 Money*QuotaPerUnit）
2. daily_quota 控制每日最大消耗上限
3. 次日凌晨 Cron 重置 daily_used = 0
```

---

## 收费流程

### 完整流程图

```
┌─────────────┐
│  用户访问   │
│  充值页面   │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  选择金额   │ ← 可配置充值档位: ¥50, ¥100, ¥200, ¥500
│  选择支付   │ ← Stripe / 易支付 / Creem
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ 创建订单    │ → DB: INSERT topups (status=pending)
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ 调用支付API │ → 获取支付链接
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ 跳转支付页  │ → 用户扫码/输入卡号
└──────┬──────┘
       │
       ▼
┌─────────────┐
│Webhook 回调 │ → 验证签名 + 防重放（status=pending 校验）
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ 更新订单    │ → UPDATE topups SET status=success
│ 增加额度    │ → UPDATE users SET quota = quota + amount
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ 充值成功    │ → 跳转回控制台
└─────────────┘
```

### 关键安全措施

**1. 订单防重放**
```go
// 每个订单使用唯一 trade_no
tradeNo := fmt.Sprintf("USR%dNO%s%d", userId, common.GetRandomString(6), time.Now().Unix())

// Webhook 验证 trade_no 未处理过
if topup.Status != common.TopUpStatusPending {
    return // 已处理，忽略（幂等）
}
```

**2. 签名验证**
```go
// Stripe Webhook 签名验证
event, err := webhook.ConstructEvent(body, signature, webhookSecret)

// 易支付签名验证
verifyInfo, err := client.Verify(params)

// Creem 签名验证（HMAC-SHA256）
mac := hmac.New(sha256.New, []byte(secret))
```

**3. 租户隔离**
```go
// Webhook 回调中验证租户一致性
if user.TenantId != topup.TenantId {
    return errors.New("租户验证失败")
}
```

**4. 行级锁防并发**
```go
tx.Set("gorm:query_option", "FOR UPDATE").Where("trade_no = ?", tradeNo).First(&topUp)
```

---

## 多产品计费能力评估

> 本节回答：当前 lurus-api 计费系统能否支撑整个公司所有产品的计费需求？

### 当前计费模式支持状态

| 模式 | 实现状态 | 机制 |
|------|---------|------|
| 按 Token/Quota 计费 | ✅ 完整 | `PreConsumeQuota` + `PostConsumeQuota` |
| 按次计费 | ✅ 完整 | `logs.request_count` 记录每次请求 |
| 包月/包年订阅 | ✅ 完整 | `subscriptions` 表 + Cron 管理 |
| 一次性充值 | ✅ 完整 | `topups` 表，Webhook 自动到账 |
| 充值档位折扣 | ✅ 完整 | `AmountDiscount` map |
| 分组差异化定价 | ✅ 完整 | `TopupGroupRatio`，VIP 可配不同比率 |
| 订阅自动续费 | ✅ 已实现 | 余额扣费，每小时 Cron 检查（见 `subscription_cron.go`） |

### 对 AI 网关自身的结论

**✅ 可以支撑**（条件：不依赖直连官方支付宝/微信）

- 所有计费模式均已实现并具备生产级安全机制
- 国内支付通过**易支付**已可走支付宝/微信通道
- 若要官方直连（0.6% 费率），需企业资质 + 接入支付宝开放平台/微信支付商户

### 对多产品公司计费中台的结论

**❌ 不能直接支撑（当前架构限制）**

| 问题 | 当前状态 | 影响 |
|------|---------|------|
| 计费单位与 Token 强耦合 | Quota = Token 数量 | 非 AI 产品无法直接复用 |
| 无通用扣费 API | 外部服务无法调用 | 无法集成其他产品 |
| 无产品维度隔离 | `tenant_id` 是组织维度 | 跨产品统计无法细分 |
| 无统一账单视图 | 只展示 AI 调用记录 | 无法展示多产品消费明细 |
| 退款系统 | ❌ 缺失 | 无法处理投诉退款 |
| 发票/收据 | ❌ 缺失 | 企业合规需求无法满足 |

### 推荐路径

**路径 B（推荐）：保持 lurus-api 专注 AI 网关，各产品自治计费**

```
短期：
  lurus-api       → AI 网关计费（现有方案，已生产就绪）
  其他产品        → 各自接入支付 SDK（共享 Zitadel SSO 用户体系）

长期（规模扩大后）：
  lurus-billing   → 独立计费中台（统一账单、退款、对账）
                    所有产品通过内部 API 调用
```

**路径 A（可选）：扩展 lurus-api 为计费中台**

需要引入：
1. `product_id` 字段（区分不同产品）
2. 通用扣费 API（供其他服务调用）
3. 通用计量单位（非仅 Token）

改动较大，影响现有逻辑，建议在 Epic 级别规划。

---

## 配置指南

### 1. 配置支付渠道

**Root 登录 → 系统设置 → 支付配置**

#### Stripe 配置

```bash
# 1. 获取 Stripe 密钥
https://dashboard.stripe.com/apikeys

# 2. 配置 Webhook 端点（选择以下事件）
https://api.lurus.cn/api/pay/stripe
✅ checkout.session.completed
✅ payment_intent.succeeded
```

#### 易支付配置（含支付宝/微信通道）

```bash
# 从易支付服务商获取以下信息
PID: 12345
密钥: xxxxx
API地址: https://pay.example.com

# 回调地址
Notify URL: https://api.lurus.cn/api/user/epay/notify
Return URL: https://api.lurus.cn/console/log
```

---

### 2. 配置充值档位

**系统设置 → 充值配置**

```json
{
  "amount_options": [50, 100, 200, 500],
  "amount_discount": {
    "500": 0.95,
    "1000": 0.90
  },
  "min_topup": 10
}
```

**字段说明：**
- `amount_options`: 充值档位（元）
- `amount_discount`: 充值档位折扣（充值越多折扣越大）
- `min_topup`: 最低充值金额

---

### 3. 配置订阅套餐

**Root 登录 → 系统设置 → 订阅套餐管理**

```json
[
  {
    "code": "monthly",
    "name": "月付套餐",
    "days": 30,
    "price": 59.9,
    "currency": "CNY",
    "daily_quota": 1000000,
    "total_quota": 50000000,
    "base_group": "monthly",
    "fallback_group": "weekly",
    "enabled": true
  }
]
```

---

### 4. 自动续费说明

- 自动续费每小时运行一次（`autoRenewalProcessorWithContext`）
- 检查 24h 内到期 + `auto_renew=true` 的订阅
- 从用户 `quota` 余额中扣除续费费用（`plan.Price * QuotaPerUnit`）
- 扣费同时补充新期 `TotalQuota` 到余额（原子事务）
- 余额不足时：记录警告日志，不续费（待接入邮件通知）

---

## 常见问题

### Q1: 用户充值后多久到账？

**A:** 取决于支付方式：
- **Stripe**: 即时到账（Webhook 1-3秒）
- **易支付（支付宝/微信通道）**: 即时到账（Webhook 1-5秒）
- **Creem**: 即时到账（Webhook 实时）
- **银行转账**: 管理员人工核对后到账

---

### Q2: 订阅到期后会自动续费吗？

**A:** 取决于用户设置：
- **`auto_renew = true`**: 到期前 24 小时尝试从 quota 余额扣费续费
- **`auto_renew = false`**: 到期后失效，不续费

**自动续费逻辑（`subscription_cron.go: processOneAutoRenewal`）：**
```go
// 1. 计算续费成本（CNY 转换为 quota 单位）
renewalCost := int(plan.Price * common.QuotaPerUnit)

// 2. 余额足够：原子事务扣费 + 延长有效期 + 补充新期额度
DB.Transaction(func(tx *gorm.DB) error {
    tx.Model(&User{}).Where("id = ?", userId).
        Update("quota", gorm.Expr("quota + ?", plan.TotalQuota - renewalCost))
    tx.Model(&sub).Update("expires_at", sub.ExpiresAt.AddDate(0, 0, plan.Days))
    return nil
})

// 3. 余额不足：记录日志，待邮件通知（TODO）
```

---

### Q3: 用户可以退款吗？

**A:** 当前系统**无退款模块**（计划在独立计费中台实现）。

临时处理方案：
```
1. 用户提交工单说明退款原因
2. 管理员在 Stripe/易支付后台手动退款
3. 管理员通过 API 扣减用户 quota：
   Root 登录 → 用户管理 → 选择用户 → 调整额度
```

---

### Q4: 如何防止用户恶意充值后申请退款？

**A:** 多重防护措施：

**1. 设置冷静期（需手动实施）**
```go
// 充值后 7 天内使用超过 50% 不可退款
if time.Since(topup.CreateTime) < 7*24*time.Hour &&
   user.UsedQuota > topup.Amount*0.5 {
    return errors.New("已使用超过50%，不可退款")
}
```

**2. 风控监控（需接入告警系统）**
```yaml
自动触发审核:
  - 单笔充值 > ¥1000
  - 24小时内充值 > 3 次
  - 充值后 1 小时内使用超过 80%
```

---

### Q5: 支持企业对公转账吗？

**A:** 支持，需要管理员手动处理：

```
1. 用户提交工单 → 提供转账凭证
2. 管理员验证到账
3. 管理员补单：Root 登录 → 充值管理 → 手动补单（输入订单号）
   或调整用户额度：用户管理 → 选择用户 → 调整额度
```

---

### Q6: 如何为大客户定制计费方式？

**A:** 三种方式：

**方式 1：兑换码**
```
Root 登录 → 兑换码管理 → 批量生成（指定额度 + 有效期）
发送给大客户 → 客户自行兑换
```

**方式 2：直接调整额度**
```
Root 登录 → 用户管理 → 选择用户 → 调整额度
```

**方式 3：内部订阅（不走支付）**
```go
// 管理员通过 Internal API 直接激活订阅
POST /api/v2/{tenant}/internal/subscriptions
{
  "user_id": 123,
  "plan_code": "monthly",
  "days": 90,
  "reason": "大客户合作"
}
```

---

### Q7: 多产品如何财务对账？

**A:** 通过 `tenant_id` 维度统计（当前能力）：

```sql
-- 按产品统计充值收入
SELECT
  tu.tenant_id AS product,
  COUNT(DISTINCT tu.user_id) AS paying_users,
  SUM(tu.money) AS revenue_cny,
  COUNT(*) AS order_count
FROM topups tu
WHERE tu.status = 'success'
  AND tu.create_time >= '2026-02-01'
GROUP BY tu.tenant_id;

-- 按产品统计消耗（AI 调用）
SELECT
  l.tenant_id AS product,
  SUM(l.quota) AS quota_consumed,
  COUNT(*) AS api_calls
FROM logs l
WHERE l.created_at >= '2026-02-01'
GROUP BY l.tenant_id;
```

> **注意**：当前 `tenant_id` 是组织/产品维度，非独立产品账户维度。
> 多产品统一对账账单功能在规划中的 `lurus-billing` 服务中实现。

---

## 总结

### 计费能力矩阵

| 功能 | 状态 | 备注 |
|------|------|------|
| 按量充值 | ✅ | Stripe/易支付/Creem |
| 包月订阅 | ✅ | 四档套餐，可自定义 |
| 订阅自动续费 | ✅ | 余额扣费，24h 前触发 |
| 国内支付（易支付通道） | ✅ | 支付宝/微信/银联 |
| 国际支付 | ✅ | Stripe + Creem |
| 兑换码 | ✅ | 批量发放 |
| 多产品计费中台 | ❌ | 需独立服务 |
| 退款系统 | ❌ | 计划中 |
| 发票/收据 | ❌ | 计划中 |

### 多产品接入推荐方案

- **认证**: Zitadel 统一 SSO（账号共享）
- **AI 计费**: lurus-api 共享钱包（余额通用，按 tenant_id 分组统计）
- **非 AI 产品**: 各自独立处理支付，共享 Zitadel 用户 ID
- **未来**: 独立 `lurus-billing` 服务作为统一计费中台

---

**文档维护**: Lurus 技术团队
**最后更新**: 2026-02-25
