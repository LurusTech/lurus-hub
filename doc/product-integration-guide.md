# 产品接入指南 - Lurus 统一登录平台

## 目录
- [概述](#概述)
- [架构说明](#架构说明)
- [接入步骤](#接入步骤)
- [前端集成](#前端集成)
- [后端集成](#后端集成)
- [测试验证](#测试验证)
- [常见问题](#常见问题)

---

## 概述

Lurus 平台提供统一的用户认证和 AI 服务网关：
- **认证中心**: Zitadel OAuth 2.0 / OIDC
- **AI 网关**: 统一的 OpenAI 兼容 API
- **计费系统**: 集中的额度管理和订阅

### 接入后您的产品将获得

✅ **统一身份认证**：用户一次注册，所有产品通用
✅ **AI 能力**：无需对接各 AI 厂商，统一调用 OpenAI 格式 API
✅ **计费托管**：自动扣费，用户自助充值和订阅
✅ **管理后台**：用量监控、API Token 管理

---

## 架构说明

### 用户登录流程

```
[产品前端]
    ↓ 点击"登录"
https://api.lurus.cn/login/{product-slug}
    ↓
[Zitadel 登录页]
    ↓ 用户输入邮箱密码（或 SSO）
[OAuth 授权]
    ↓
[Lurus 回调处理]
    ↓ 创建/关联租户
    ↓ 分配 Session
[Lurus 控制台]
    ↓ 用户创建 API Token
[产品后端] ← sk-xxx ← [用户复制 Token]
    ↓
[AI API 调用] POST /v1/chat/completions
```

### 数据隔离

| 资源 | 隔离级别 |
|------|---------|
| 用户账号 | **共享**（SSO） |
| API Token | 按产品隔离 |
| 使用日志 | 按产品隔离 |
| 额度/计费 | 按产品隔离 |
| AI 渠道配置 | 共享（可选按产品定制） |

---

## 接入步骤

### 步骤 1：在 Zitadel 创建 Application

**登录 Zitadel 管理后台：**
- URL: https://auth.lurus.cn/ui/console
- 账号：联系管理员获取

**操作步骤：**

1. 进入 Organization: `lurus`
2. 点击 **Applications** → **New**
3. 填写信息：
   - **Name**: `{产品英文名}` (例如: `product-b`)
   - **Type**: `Web`
   - **Authentication Method**: `PKCE`

4. **Redirect URIs**（重要）：
   ```
   https://api.lurus.cn/api/v2/oauth/callback
   ```

5. **Post Logout Redirect URIs**：
   ```
   https://api.lurus.cn
   https://{你的产品域名}
   ```

6. **Grant Types**：勾选
   - ✅ Authorization Code
   - ✅ Refresh Token

7. 点击 **Create**

8. **记录 Client ID**（在 Application 详情页）:
   ```
   例如: 234567890123456789@lurus
   ```

---

### 步骤 2：在 Lurus 注册租户

联系 Lurus 管理员提供以下信息：

```yaml
产品信息:
  slug: product-b              # 产品标识（小写字母、数字、连字符）
  name: Product B              # 产品显示名称
  zitadel_org_id: xxxx         # Zitadel Organization ID（通常是 lurus）
  zitadel_client_id: xxxx      # 上一步获得的 Client ID
  admin_email: admin@product-b.com  # 产品管理员邮箱
```

管理员将执行：
```sql
-- 创建租户（管理员操作，仅供参考）
INSERT INTO tenants (slug, name, zitadel_org_id, zitadel_client_id, status, created_at)
VALUES
  ('product-b', 'Product B', '{org_id}', '{client_id}', 1, NOW());

-- 为产品管理员分配权限
INSERT INTO tenant_admins (tenant_id, user_email, role)
VALUES
  ((SELECT id FROM tenants WHERE slug = 'product-b'), 'admin@product-b.com', 'owner');
```

**注册完成后您将收到：**
- ✅ 登录入口 URL: `https://api.lurus.cn/login/product-b`
- ✅ 测试账号（可选）

---

### 步骤 3：配置产品环境变量

在您的产品后端配置：

```bash
# .env
LURUS_API_BASE_URL=https://api.lurus.cn
LURUS_API_KEY=sk-xxxxxxxxxxxx  # 从控制台获取
```

---

## 前端集成

### 方式 1：直接链接（推荐）

**登录按钮：**
```html
<a href="https://api.lurus.cn/login/product-b" class="btn-login">
  使用 Lurus 账号登录
</a>
```

**特点：**
- ✅ 最简单，无需任何代码
- ✅ 用户登录后进入 Lurus 控制台
- ✅ 用户手动复制 API Token 到您的产品

---

### 方式 2：嵌入式集成（进阶）

**适用场景：** 需要用户在您的产品内完成登录，不跳转到 Lurus 控制台

#### React 示例

```jsx
import { useState, useEffect } from 'react';

const LurusAuth = () => {
  const [apiKey, setApiKey] = useState(localStorage.getItem('lurus_api_key'));

  const handleLogin = () => {
    // 保存当前页面 URL，登录后返回
    sessionStorage.setItem('return_url', window.location.href);

    // 跳转到 Lurus 登录
    window.location.href = 'https://api.lurus.cn/login/product-b';
  };

  const handleLogout = async () => {
    // 调用 Lurus 登出 API
    await fetch('https://api.lurus.cn/api/v2/oauth/logout', {
      method: 'POST',
      credentials: 'include'
    });

    localStorage.removeItem('lurus_api_key');
    setApiKey(null);
  };

  // 检查用户登录状态
  useEffect(() => {
    const checkSession = async () => {
      try {
        const res = await fetch('https://api.lurus.cn/api/v2/auth/session-info', {
          credentials: 'include'
        });
        const data = await res.json();

        if (data.success && data.data.id) {
          // 用户已登录 Lurus，引导创建 Token
          console.log('User logged in:', data.data);
        }
      } catch (error) {
        console.error('Session check failed:', error);
      }
    };

    checkSession();
  }, []);

  return (
    <div>
      {!apiKey ? (
        <button onClick={handleLogin}>登录 Lurus</button>
      ) : (
        <div>
          <p>API Key: {apiKey.substring(0, 10)}...</p>
          <button onClick={handleLogout}>登出</button>
        </div>
      )}
    </div>
  );
};

export default LurusAuth;
```

#### Vue 示例

```vue
<template>
  <div>
    <button v-if="!isLoggedIn" @click="login">登录 Lurus</button>
    <div v-else>
      <p>已登录: {{ userName }}</p>
      <button @click="logout">登出</button>
    </div>
  </div>
</template>

<script>
export default {
  data() {
    return {
      isLoggedIn: false,
      userName: '',
      apiKey: localStorage.getItem('lurus_api_key') || ''
    };
  },

  mounted() {
    this.checkSession();
  },

  methods: {
    login() {
      sessionStorage.setItem('return_url', window.location.href);
      window.location.href = 'https://api.lurus.cn/login/product-b';
    },

    async logout() {
      await fetch('https://api.lurus.cn/api/v2/oauth/logout', {
        method: 'POST',
        credentials: 'include'
      });

      this.isLoggedIn = false;
      this.userName = '';
      localStorage.removeItem('lurus_api_key');
    },

    async checkSession() {
      try {
        const res = await fetch('https://api.lurus.cn/api/v2/auth/session-info', {
          credentials: 'include'
        });
        const data = await res.json();

        if (data.success && data.data.id) {
          this.isLoggedIn = true;
          this.userName = data.data.display_name || data.data.username;
        }
      } catch (error) {
        console.error('Session check failed:', error);
      }
    }
  }
};
</script>
```

---

### 方式 3：回调页面（自动获取 Token）

**适用场景：** 用户登录后自动跳回您的产品，无需手动复制 Token

**实现步骤：**

1. **配置自定义回调 URL**（需要管理员协助）：
   ```sql
   -- 在 tenants 表添加自定义回调
   UPDATE tenants
   SET custom_redirect_url = 'https://yourapp.com/auth/callback'
   WHERE slug = 'product-b';
   ```

2. **在您的产品创建回调页面** (`/auth/callback`)：

```javascript
// /auth/callback 页面
const urlParams = new URLSearchParams(window.location.search);
const token = urlParams.get('token');  // 从 URL 获取临时 token

if (token) {
  // 用临时 token 换取 API Key
  fetch('https://api.lurus.cn/api/v2/product-b/auth/exchange-token', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ temp_token: token })
  })
  .then(res => res.json())
  .then(data => {
    if (data.success) {
      localStorage.setItem('lurus_api_key', data.api_key);

      // 跳回登录前的页面
      const returnUrl = sessionStorage.getItem('return_url') || '/';
      window.location.href = returnUrl;
    }
  });
}
```

---

## 后端集成

### Node.js 示例

```javascript
// config.js
const LURUS_API_BASE = process.env.LURUS_API_BASE_URL || 'https://api.lurus.cn';
const LURUS_API_KEY = process.env.LURUS_API_KEY;

// AI 服务调用
const callAI = async (messages) => {
  const response = await fetch(`${LURUS_API_BASE}/v1/chat/completions`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${LURUS_API_KEY}`
    },
    body: JSON.stringify({
      model: 'gpt-4o',
      messages: messages,
      temperature: 0.7
    })
  });

  const data = await response.json();

  if (!response.ok) {
    throw new Error(`Lurus API Error: ${data.error?.message || 'Unknown error'}`);
  }

  return data.choices[0].message.content;
};

// 使用示例
app.post('/api/chat', async (req, res) => {
  try {
    const { message } = req.body;

    const aiResponse = await callAI([
      { role: 'user', content: message }
    ]);

    res.json({ success: true, response: aiResponse });
  } catch (error) {
    console.error('AI call failed:', error);
    res.status(500).json({ success: false, error: error.message });
  }
});
```

---

### Python 示例

```python
# config.py
import os
from openai import OpenAI

# 使用 OpenAI SDK，只需修改 base_url
client = OpenAI(
    api_key=os.getenv("LURUS_API_KEY"),
    base_url="https://api.lurus.cn/v1"
)

def call_ai(messages: list) -> str:
    """调用 Lurus AI 服务"""
    try:
        response = client.chat.completions.create(
            model="gpt-4o",
            messages=messages,
            temperature=0.7
        )
        return response.choices[0].message.content
    except Exception as e:
        print(f"AI call failed: {e}")
        raise

# 使用示例
from flask import Flask, request, jsonify

app = Flask(__name__)

@app.route('/api/chat', methods=['POST'])
def chat():
    try:
        data = request.json
        message = data.get('message')

        ai_response = call_ai([
            {"role": "user", "content": message}
        ])

        return jsonify({"success": True, "response": ai_response})
    except Exception as e:
        return jsonify({"success": False, "error": str(e)}), 500
```

---

### Go 示例

```go
// lurus/client.go
package lurus

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
)

type ChatRequest struct {
    Model       string    `json:"model"`
    Messages    []Message `json:"messages"`
    Temperature float64   `json:"temperature,omitempty"`
}

type Message struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

type ChatResponse struct {
    Choices []struct {
        Message Message `json:"message"`
    } `json:"choices"`
    Error *struct {
        Message string `json:"message"`
    } `json:"error,omitempty"`
}

type Client struct {
    baseURL string
    apiKey  string
    http    *http.Client
}

func NewClient() *Client {
    return &Client{
        baseURL: os.Getenv("LURUS_API_BASE_URL"),
        apiKey:  os.Getenv("LURUS_API_KEY"),
        http:    &http.Client{},
    }
}

func (c *Client) Chat(messages []Message) (string, error) {
    reqBody := ChatRequest{
        Model:       "gpt-4o",
        Messages:    messages,
        Temperature: 0.7,
    }

    jsonData, _ := json.Marshal(reqBody)

    req, err := http.NewRequest("POST", c.baseURL+"/v1/chat/completions", bytes.NewBuffer(jsonData))
    if err != nil {
        return "", err
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+c.apiKey)

    resp, err := c.http.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)

    var chatResp ChatResponse
    if err := json.Unmarshal(body, &chatResp); err != nil {
        return "", err
    }

    if chatResp.Error != nil {
        return "", fmt.Errorf("API error: %s", chatResp.Error.Message)
    }

    return chatResp.Choices[0].Message.Content, nil
}

// 使用示例
func main() {
    client := lurus.NewClient()

    response, err := client.Chat([]lurus.Message{
        {Role: "user", Content: "Hello, AI!"},
    })

    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(response)
}
```

---

## 测试验证

### 1. 测试登录流程

**手动测试：**
```bash
# 1. 访问登录页面
https://api.lurus.cn/login/product-b

# 2. 输入测试账号
email: test@product-b.com
password: Test@123456

# 3. 验证跳转到控制台
https://api.lurus.cn/console
```

**预期结果：**
- ✅ 成功跳转到 Zitadel 登录页
- ✅ 登录后返回 Lurus 控制台
- ✅ 页面右上角显示用户邮箱
- ✅ 用户分组显示为 `product-b`（或 `default`）

---

### 2. 测试 API Token 创建

**在 Lurus 控制台：**
1. 登录后点击左侧 **令牌管理**
2. 点击 **创建令牌**
3. 填写名称：`Product B Test Token`
4. 点击 **提交**
5. 复制生成的 Token：`sk-xxxxxxxxxxxx`

---

### 3. 测试 AI API 调用

**cURL 测试：**
```bash
curl -X POST https://api.lurus.cn/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-xxxxxxxxxxxx" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [
      {"role": "user", "content": "Hello, this is a test from Product B"}
    ]
  }'
```

**预期返回：**
```json
{
  "id": "chatcmpl-xxx",
  "object": "chat.completion",
  "created": 1234567890,
  "model": "gpt-4o-mini",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Hello! I received your test message from Product B. How can I assist you today?"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 15,
    "completion_tokens": 20,
    "total_tokens": 35
  }
}
```

---

### 4. 测试用量查询

**查询当前用户余额：**
```bash
curl https://api.lurus.cn/api/v2/product-b/user/me \
  -H "Authorization: Bearer sk-xxxxxxxxxxxx"
```

**返回示例：**
```json
{
  "success": true,
  "data": {
    "id": 123,
    "username": "user123",
    "quota": 10000000,
    "used_quota": 5000,
    "remaining_quota": 9995000,
    "daily_quota": {
      "limit": 100000,
      "used": 5000,
      "remaining": 95000
    },
    "subscription": {
      "plan_code": "monthly-basic",
      "status": "active",
      "expires_at": "2026-03-10T00:00:00Z"
    }
  }
}
```

---

## 常见问题

### Q1: 用户登录后看不到我的产品？

**A:** Lurus 是 AI 服务网关，不是产品集成平台。用户登录后：
1. 进入 Lurus 控制台
2. 创建 API Token
3. 将 Token 配置到您的产品中
4. 您的产品通过 Token 调用 AI 服务

如需无缝体验，请使用**方式 3：回调页面集成**。

---

### Q2: 多个产品的用户数据会混在一起吗？

**A:** 不会。虽然用户账号共享（SSO），但业务数据完全隔离：
- 每个产品有独立的 `tenant_id`
- API Token 绑定到特定租户
- 使用日志、计费记录按租户隔离
- 用户在产品 A 的 Token 无法在产品 B 使用

---

### Q3: 如何为特定产品配置专属 AI 模型？

**A:** 联系管理员配置渠道权限：
```sql
-- 为产品 B 分配专属渠道
INSERT INTO tenant_channels (tenant_id, channel_id, priority)
VALUES
  ((SELECT id FROM tenants WHERE slug = 'product-b'),
   (SELECT id FROM channels WHERE name = 'OpenAI-ProductB'),
   100);
```

---

### Q4: 用户额度不足时会发生什么？

**A:** API 调用返回 402 错误：
```json
{
  "error": {
    "message": "Insufficient quota. Please top up your account.",
    "type": "insufficient_quota",
    "code": "quota_exceeded"
  }
}
```

用户需要：
1. 登录 Lurus 控制台
2. 进入"钱包管理"充值
3. 或订阅套餐

---

### Q5: 如何限制用户在我的产品中的调用频率？

**A:** 两种方式：

**方式 1：使用 Lurus 的日限额**（已内置）
- 每个用户有 `daily_quota`
- 超出后当日禁止调用

**方式 2：在您的产品中实现**
```javascript
// 在您的后端添加限流
const rateLimit = require('express-rate-limit');

const aiLimiter = rateLimit({
  windowMs: 60 * 1000, // 1分钟
  max: 10,             // 最多10次请求
  message: 'Too many AI requests, please try again later.'
});

app.post('/api/chat', aiLimiter, async (req, res) => {
  // 调用 Lurus API
});
```

---

### Q6: 如何实现白标（去掉 Lurus 品牌）？

**A:** 当前暂不支持完全白标，但可以：

1. **自定义登录页样式**（需要管理员协助修改 Zitadel 主题）
2. **使用方式 3 回调集成**：用户登录后直接返回您的产品
3. **隐藏 Lurus 控制台**：用户不感知 Lurus 存在，Token 自动管理

---

### Q7: 支持哪些 AI 模型？

**A:** 完整列表请访问：https://api.lurus.cn/console/models

常用模型：
- OpenAI: `gpt-4o`, `gpt-4o-mini`, `gpt-3.5-turbo`
- Anthropic: `claude-3-5-sonnet`, `claude-3-opus`
- Google: `gemini-pro`, `gemini-1.5-pro`
- 国内: `qwen-max`, `glm-4`, `deepseek-chat`

---

### Q8: 如何获取技术支持？

**支持渠道：**
- 📧 邮箱: support@quantumnous.com
- 📖 文档: https://docs.lurus.cn
- 💬 企业微信群：联系管理员拉群

**紧急问题：**
- 🔴 生产环境故障：直接致电管理员
- 🟡 账单/充值问题：邮件联系财务
- 🟢 功能咨询：企业微信群

---

## 附录

### A. API 完整端点列表

| 端点 | 方法 | 说明 |
|------|------|------|
| `/v1/chat/completions` | POST | 对话模型调用 |
| `/v1/embeddings` | POST | 文本向量化 |
| `/v1/images/generations` | POST | 图片生成 |
| `/v1/audio/transcriptions` | POST | 语音转文字 |
| `/api/v2/{tenant}/user/me` | GET | 获取用户信息 |
| `/api/v2/{tenant}/tokens` | GET | 查询 Token 列表 |
| `/api/v2/{tenant}/tokens` | POST | 创建 Token |
| `/api/v2/{tenant}/logs` | GET | 查询使用日志 |
| `/api/v2/{tenant}/billing/topup` | POST | 发起充值 |

### B. 错误码对照表

| HTTP 状态码 | 错误类型 | 说明 | 处理建议 |
|------------|---------|------|---------|
| 401 | `invalid_api_key` | API Key 无效 | 检查 Token 是否正确 |
| 402 | `insufficient_quota` | 额度不足 | 提示用户充值 |
| 429 | `rate_limit_exceeded` | 超出速率限制 | 稍后重试 |
| 500 | `upstream_error` | AI 服务商故障 | 重试或切换模型 |
| 503 | `service_unavailable` | Lurus 维护中 | 等待恢复 |

### C. Webhook 通知（可选）

如需接收用户充值、订阅等事件通知，请提供 Webhook URL：

```json
POST https://yourapp.com/webhooks/lurus
Content-Type: application/json

{
  "event": "user.quota.recharged",
  "tenant_slug": "product-b",
  "user_id": 123,
  "data": {
    "amount": 100000,
    "quota_before": 5000,
    "quota_after": 105000,
    "timestamp": "2026-02-10T10:30:00Z"
  },
  "signature": "sha256=xxxxx"  // 用于验证请求真实性
}
```

---

## 更新日志

| 日期 | 版本 | 变更 |
|------|------|------|
| 2026-02-10 | v1.0 | 初始版本 |

---

**文档维护**: Lurus 技术团队
**最后更新**: 2026-02-10
**反馈**: support@quantumnous.com
