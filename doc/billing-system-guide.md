# Lurus 计费系统说明

## 目录
- [系统架构](#系统架构)
- [支持的支付方式](#支持的支付方式)
- [计费模式](#计费模式)
- [收费流程](#收费流程)
- [多产品计费方案](#多产品计费方案)
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
| **额度管理** | 追踪用户剩余额度、日限额 | `users.quota`, `users.used_quota` |
| **扣费系统** | AI 调用后自动扣费 | `logs` |
| **兑换码** | 批量发放额度 | `redemptions` |

---

## 支持的支付方式

### 已集成支付渠道

| 支付方式 | 代码标识 | 适用地区 | 手续费 | 状态 |
|---------|---------|---------|--------|------|
| **Stripe** | `stripe` | 🌍 全球 | 2.9% + $0.30 | ✅ 生产可用 |
| **易支付 (Epay)** | `epay` | 🇨🇳 中国 | 自定义 | ✅ 生产可用 |
| **Creem** | `creem` | 🌍 全球 | 3.5% | ✅ 生产可用 |
| **支付宝** | `alipay` | 🇨🇳 中国 | 0.6% | 🚧 集成中 |
| **微信支付** | `wechat` | 🇨🇳 中国 | 0.6% | 🚧 集成中 |

### 配置方式

**通过系统设置 → 支付配置**

```go
// 每种支付方式的配置示例

// 1. Stripe
STRIPE_SECRET_KEY=sk_live_xxxxx
STRIPE_WEBHOOK_SECRET=whsec_xxxxx

// 2. Epay (易支付)
EPAY_URL=https://pay.example.com
EPAY_PID=12345
EPAY_KEY=xxxxx

// 3. Creem
CREEM_API_KEY=xxxxx
CREEM_SECRET=xxxxx

// 4. 支付宝
ALIPAY_APP_ID=xxxxx
ALIPAY_PRIVATE_KEY=xxxxx
ALIPAY_PUBLIC_KEY=xxxxx

// 5. 微信支付
WECHAT_APP_ID=xxxxx
WECHAT_MCH_ID=xxxxx
WECHAT_API_KEY=xxxxx
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
用户 → 选择充值金额 (¥100) → 支付 → 到账额度 (100万 tokens)
```

**换算关系（可配置）：**
```
¥1 = 10,000 tokens
¥100 = 1,000,000 tokens
```

**API 端点：**
```bash
# 1. 创建充值订单
POST /api/v2/{tenant}/billing/topup
{
  "amount": 1000000,  # 额度数量
  "money": 100        # 支付金额（元）
}

# 2. 发起支付
POST /api/v2/{tenant}/billing/pay
{
  "trade_no": "xxx",
  "payment_method": "stripe"  # stripe | epay | creem
}

# 响应: { "payment_url": "https://stripe.com/checkout/..." }

# 3. 用户跳转支付 → 支付完成 → Webhook 回调 → 自动到账
```

---

### 模式 2：订阅制

**特点：**
- ✅ 周期性自动扣费（月/年）
- ✅ 包含固定额度 + 日限额
- ✅ 适合高频用户

**套餐示例：**

| 套餐 | 价格 | 总额度 | 日限额 | 有效期 |
|------|------|--------|--------|--------|
| 基础版 | ¥99/月 | 100万 tokens | 5万/天 | 30天 |
| 专业版 | ¥299/月 | 500万 tokens | 20万/天 | 30天 |
| 企业版 | ¥999/月 | 2000万 tokens | 100万/天 | 30天 |

**订阅流程：**
```
用户 → 选择套餐 → 支付 → 订阅生效 → 每月自动续费
```

**API 端点：**
```bash
# 1. 获取套餐列表
GET /api/v2/{tenant}/billing/plans

# 2. 创建订阅
POST /api/v2/{tenant}/billing/subscriptions
{
  "plan_code": "monthly-basic",
  "payment_method": "stripe",
  "auto_renew": true
}

# 3. 取消订阅
DELETE /api/v2/{tenant}/billing/subscriptions/{id}
```

---

### 模式 3：混合模式（推荐）

**特点：**
- ✅ 订阅 + 充值并存
- ✅ 订阅提供基础额度，充值用于超额部分
- ✅ 适合所有场景

**计算逻辑：**
```
1. 优先消耗订阅额度（日限额内）
2. 日限额用完后，消耗充值余额
3. 次日重置日限额
```

**示例：**
```
用户订阅"专业版"（日限额 20万）+ 充值 50万

Day 1:
  - 使用 15万 → 消耗订阅日限额 (剩余 5万)

Day 2:
  - 日限额重置为 20万
  - 使用 25万 → 消耗订阅 20万 + 充值 5万

用户总剩余:
  - 订阅额度: 500万 - 35万 = 465万
  - 充值余额: 50万 - 5万 = 45万
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
│  选择支付   │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ 创建订单    │ → DB: INSERT topups (status=pending)
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ 调用支付API │
│ (Stripe等)  │ → 获取支付链接
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ 跳转支付页  │ → 用户扫码/输入卡号
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ 用户支付    │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│Webhook 回调 │ → POST /api/webhooks/stripe
│验证签名      │ → 验证 trade_no, amount
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
│ 发送通知    │ → 邮件/站内信
└─────────────┘
```

### 关键安全措施

**1. 订单防重放**
```go
// 每个订单使用唯一 trade_no
tradeNo := common.GetRandomString(32)

// Webhook 验证 trade_no 未处理过
if topup.Status != common.TopUpStatusPending {
    return // 已处理，忽略
}
```

**2. 金额验证**
```go
// Webhook 中必须验证金额匹配
if webhookAmount != topup.Money {
    log.Error("Amount mismatch!")
    return
}
```

**3. 签名验证**
```go
// Stripe Webhook 签名验证
signature := r.Header.Get("Stripe-Signature")
event, err := webhook.ConstructEvent(body, signature, webhookSecret)
if err != nil {
    return // 非法请求
}
```

**4. 幂等性保证**
```sql
-- 使用事务 + 乐观锁
BEGIN;
UPDATE users SET quota = quota + 1000000
WHERE id = 123 AND version = current_version;
UPDATE topups SET status = 'success' WHERE trade_no = 'xxx' AND status = 'pending';
COMMIT;
```

---

## 多产品计费方案

### 方案对比

#### **方案 A：共享钱包（推荐）**

**特点：**
- ✅ 用户在所有产品共享一个余额
- ✅ 在产品 A 充值，可在产品 B 使用
- ✅ 统一计费，用户体验好

**实现：**
```sql
-- 用户表：所有产品共享 quota
users
  ├─ id (用户全局唯一)
  ├─ quota (总额度)
  └─ used_quota (已使用)

-- 使用日志：按产品分开记录
logs
  ├─ user_id
  ├─ tenant_id  ← 区分产品
  └─ quota_used
```

**查询示例：**
```sql
-- 查询用户在产品 B 的消费
SELECT SUM(quota_used) FROM logs
WHERE user_id = 123 AND tenant_id = (SELECT id FROM tenants WHERE slug = 'product-b');

-- 用户总余额（所有产品共享）
SELECT quota - used_quota FROM users WHERE id = 123;
```

---

#### **方案 B：独立钱包**

**特点：**
- ✅ 每个产品的余额独立
- ✅ 产品 A 的余额不能在产品 B 使用
- ✅ 适合需要严格财务隔离的场景

**实现：**
```sql
-- 用户-租户关系表：每个产品独立额度
user_tenant_quotas
  ├─ user_id
  ├─ tenant_id
  ├─ quota        ← 产品独立额度
  └─ used_quota
```

**缺点：**
- ❌ 用户体验差（需要在每个产品分别充值）
- ❌ 增加开发复杂度

---

### **推荐配置：方案 A（共享钱包）**

**理由：**
1. 用户体验最佳
2. 降低用户充值门槛
3. 提高资金利用率
4. 便于用户在产品间流转

**特殊需求：**
如果某产品需要独立定价，可通过**价格系数**实现：

```go
// 产品 B 使用额度 = 实际消耗 × 价格系数
tenants
  ├─ slug: product-b
  └─ pricing_multiplier: 1.5  ← 产品 B 的 AI 调用价格是基准的 1.5 倍

// 扣费时
quotaToDeduct := actualUsage * tenant.PricingMultiplier
```

---

## 配置指南

### 1. 配置支付渠道

**Root 登录 → 系统设置 → 支付配置**

#### Stripe 配置

```bash
# 1. 获取 Stripe 密钥
https://dashboard.stripe.com/apikeys

# 2. 在系统设置中填入
Secret Key: sk_live_xxxxx
Webhook Secret: whsec_xxxxx

# 3. 配置 Webhook 端点
https://api.lurus.cn/api/webhooks/stripe

# 4. 选择事件
✅ checkout.session.completed
✅ payment_intent.succeeded
```

#### 易支付配置

```bash
# 1. 从易支付服务商获取
PID: 12345
密钥: xxxxx
API地址: https://pay.example.com

# 2. 配置回调地址
Notify URL: https://api.lurus.cn/api/webhooks/epay
Return URL: https://api.lurus.cn/console/topup?payment=success
```

---

### 2. 配置充值档位

**系统设置 → 充值配置**

```json
{
  "topup_amounts": [
    { "money": 10, "quota": 100000, "bonus": 0 },
    { "money": 50, "quota": 500000, "bonus": 10000 },
    { "money": 100, "quota": 1000000, "bonus": 50000 },
    { "money": 500, "quota": 5000000, "bonus": 500000 }
  ],
  "min_topup": 10,
  "max_topup": 10000
}
```

**字段说明：**
- `money`: 支付金额（元）
- `quota`: 到账额度（tokens）
- `bonus`: 赠送额度（可选）

---

### 3. 配置订阅套餐

**系统设置 → 订阅管理 → 新建套餐**

```sql
-- 示例：创建月付套餐
INSERT INTO subscription_plans (
  code, name, price, quota, daily_quota, duration_days, auto_renew
) VALUES (
  'monthly-basic',
  '基础版月付',
  99.00,
  1000000,    -- 总额度 100万
  50000,      -- 日限额 5万
  30,         -- 有效期 30天
  true        -- 支持自动续费
);
```

---

### 4. 配置价格系数（可选）

**适用场景：** 不同产品使用不同定价

```sql
-- 产品 B 的 AI 调用价格为基准价的 1.2 倍
UPDATE tenants
SET pricing_multiplier = 1.2
WHERE slug = 'product-b';
```

**效果：**
```
基准价格: GPT-4o = ¥0.1/1000 tokens
产品 B 价格: GPT-4o = ¥0.12/1000 tokens
```

---

## 常见问题

### Q1: 用户充值后多久到账？

**A:** 取决于支付方式：
- **Stripe**: 即时到账（Webhook 1-3秒）
- **易支付**: 即时到账（Webhook 1-5秒）
- **支付宝**: 即时到账（Webhook 实时）
- **银行转账**: 人工核对后到账（1-24小时）

---

### Q2: 订阅到期后会自动续费吗？

**A:** 取决于用户设置：
- **auto_renew = true**: 到期前 3 天自动扣费续费
- **auto_renew = false**: 到期后失效，不自动续费

**自动续费逻辑：**
```go
// 每天凌晨定时任务检查即将到期的订阅
func renewSubscriptions() {
    // 查询 3 天内到期 + 开启自动续费的订阅
    expiringSubs := getExpiringSubscriptions(3 days)

    for sub := range expiringSubs {
        // 从用户余额扣费（充值余额，非订阅额度）
        if user.Quota >= sub.Plan.Price {
            user.Quota -= sub.Plan.Price
            sub.ExpiresAt = sub.ExpiresAt.Add(30 days)
            sub.Quota = sub.Plan.Quota  // 重置额度
        } else {
            // 余额不足，发送通知
            sendEmail(user, "订阅即将到期，余额不足")
        }
    }
}
```

---

### Q3: 用户可以退款吗？

**A:** 取决于您的退款政策：

**建议策略：**
```yaml
充值退款:
  - 未使用的额度: ✅ 可退款（扣除手续费）
  - 已使用的额度: ❌ 不可退款

订阅退款:
  - 7天内未使用: ✅ 全额退款
  - 已使用部分: ❌ 按比例退款
```

**实现退款：**
```sql
-- 1. 计算可退金额
SELECT
  (quota - used_quota) / 10000 AS refundable_cny,
  used_quota / 10000 AS non_refundable_cny
FROM users WHERE id = 123;

-- 2. 创建退款订单
INSERT INTO refunds (user_id, amount, status)
VALUES (123, 50.00, 'pending');

-- 3. 扣减用户额度
UPDATE users SET quota = used_quota WHERE id = 123;

-- 4. 通过支付平台退款（Stripe Refund API）
```

---

### Q4: 如何防止用户恶意充值后申请退款？

**A:** 多重防护措施：

**1. 设置冷静期**
```go
// 充值后 7 天内使用超过 50% 不可退款
if time.Since(topup.CreateTime) < 7*24*time.Hour &&
   user.UsedQuota > topup.Amount * 0.5 {
    return errors.New("已使用超过50%，不可退款")
}
```

**2. 限制退款次数**
```go
// 每个用户每年最多退款 3 次
refundCount := getRefundCount(userID, thisYear)
if refundCount >= 3 {
    return errors.New("已达退款次数上限")
}
```

**3. 黑名单机制**
```sql
-- 标记恶意用户
UPDATE users SET risk_level = 'high' WHERE id IN (
  SELECT user_id FROM refunds
  GROUP BY user_id
  HAVING COUNT(*) > 5
);
```

---

### Q5: 支持企业对公转账吗？

**A:** 支持，需要管理员手动处理：

**流程：**
```
1. 用户提交工单 → 提供转账凭证
2. 管理员验证到账
3. 手动增加额度:

   Root 登录 → 用户管理 → 选择用户 → 调整额度

   或使用 API:
   POST /api/admin/users/{id}/quota
   {
     "amount": 1000000,
     "reason": "企业对公转账 - 发票号 xxx"
   }
```

---

### Q6: 如何为大客户定制计费方式？

**A:** 三种方式：

**方式 1：直接分配额度**
```sql
-- 给企业用户直接分配 1 亿 tokens
UPDATE users SET quota = quota + 100000000 WHERE id = 大客户ID;
```

**方式 2：创建专属套餐**
```sql
-- 创建企业专属套餐（不在前台显示）
INSERT INTO subscription_plans (code, name, price, quota, is_public)
VALUES ('enterprise-custom', '企业定制版', 9999, 100000000, false);
```

**方式 3：生成兑换码**
```sql
-- 生成 100 个兑换码，每个 100 万额度
INSERT INTO redemptions (code, quota, remain)
SELECT
  CONCAT('VIP-', UPPER(SUBSTRING(MD5(RANDOM()::text), 1, 8))),
  1000000,
  1
FROM generate_series(1, 100);
```

---

### Q7: 多产品如何财务对账？

**A:** 使用租户维度的财务报表：

```sql
-- 按产品统计收入
SELECT
  t.slug AS product,
  t.name,
  COUNT(DISTINCT tu.user_id) AS active_users,
  SUM(tu.money) AS revenue,
  SUM(tu.amount) AS quota_sold
FROM topups tu
JOIN users u ON tu.user_id = u.id
JOIN tenants t ON u.tenant_id = t.id
WHERE tu.status = 'success'
  AND tu.create_time >= '2026-02-01'
  AND tu.create_time < '2026-03-01'
GROUP BY t.slug, t.name;
```

**输出示例：**
```
product    | name        | active_users | revenue | quota_sold
-----------|-------------|--------------|---------|------------
lurus      | Lurus AI    | 1500         | ¥45,000 | 45,000,000
product-b  | Product B   | 300          | ¥9,000  | 9,000,000
product-c  | Product C   | 150          | ¥4,500  | 4,500,000
```

---

## 最佳实践

### 1. 推荐的收费策略

**对于 SaaS 产品：**
```
✅ 订阅制为主 + 充值补充
✅ 提供免费试用额度（1万 tokens）
✅ 订阅自动续费，降低流失率
```

**对于 API 平台：**
```
✅ 充值为主 + 订阅可选
✅ 按量计费，灵活透明
✅ 提供充值档位赠送（充 500 送 50）
```

---

### 2. 定价参考

| AI 模型 | OpenAI 官方价格 | Lurus 建议售价 | 毛利率 |
|---------|----------------|---------------|--------|
| GPT-4o | $2.5/1M input | ¥0.025/1k | 30% |
| GPT-4o-mini | $0.15/1M | ¥0.002/1k | 35% |
| Claude 3.5 | $3/1M | ¥0.03/1k | 30% |
| Gemini Pro | $0.5/1M | ¥0.006/1k | 40% |

**换算关系：**
```
¥1 = 10,000 tokens (针对 GPT-4o-mini)
¥100 充值 = 1,000,000 tokens = 可调用约 500 次对话
```

---

### 3. 防欺诈建议

**高风险行为监控：**
```yaml
自动触发审核:
  - 单笔充值 > ¥1000
  - 24小时内充值 > 3 次
  - 充值后 1 小时内使用超过 80%
  - 来自代理/VPN 的支付

自动冻结:
  - 同一 IP 注册超过 10 个账号
  - 使用盗用信用卡支付（Stripe 风控告警）
```

---

## 总结

### Zitadel vs Lurus 分工清晰

| 功能 | Zitadel | Lurus |
|------|---------|-------|
| 用户登录 | ✅ | - |
| 账号管理 | ✅ | - |
| 充值/订阅 | - | ✅ |
| 额度管理 | - | ✅ |
| AI 服务 | - | ✅ |

### 多产品接入推荐方案

- **认证**: Zitadel 统一 SSO（账号共享）
- **计费**: Lurus 共享钱包（余额通用）
- **数据**: 按 tenant_id 隔离

### 支持的支付方式

✅ Stripe（国际）
✅ 易支付（国内）
✅ Creem（国际）
🚧 支付宝（开发中）
🚧 微信支付（开发中）

---

**文档维护**: Lurus 技术团队
**最后更新**: 2026-02-10
**反馈**: support@quantumnous.com
