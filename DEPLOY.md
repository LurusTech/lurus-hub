# 快速部署指南

> P1 修复完成后的生产部署步骤

---

## 📋 部署前检查

- [x] ✅ P1 问题已全部修复
- [x] ✅ 代码已编译并测试通过
- [ ] ⚠️ 生成新的 Session secret
- [ ] ⚠️ 填写生产配置（secrets.prod.yaml）

---

## 🚀 快速部署（5 分钟）

### 1. 生成 Session Secret

```bash
# 生成新的随机 session secret（旧值已泄露）
openssl rand -base64 32

# 复制输出结果，下一步使用
```

### 2. 创建生产配置

```bash
cd deploy/k8s

# 复制模板
cp secrets.yaml secrets.prod.yaml

# 编辑配置
vi secrets.prod.yaml
```

**填写以下内容**（参考 `../../重要信息.md`）:

```yaml
stringData:
  # 数据库连接（使用公司标准密码）
  SQL_DSN: "postgres://lurus:LurusOps2026@100.94.177.10:30543/lurusapi?sslmode=disable"

  # Session Secret（粘贴步骤 1 生成的值）
  SESSION_SECRET: "<粘贴 openssl rand 输出>"

  # Alipay 密钥（如果已配置，联系支付团队）
  ALIPAY_PRIVATE_KEY: |
    -----BEGIN RSA PRIVATE KEY-----
    (从 Alipay 开发者控制台获取)
    -----END RSA PRIVATE KEY-----

  ALIPAY_PUBLIC_KEY: |
    -----BEGIN PUBLIC KEY-----
    (从 Alipay 开发者控制台获取)
    -----END PUBLIC KEY-----

  # Zitadel Client ID（公司标准配置）
  ZITADEL_CLIENT_ID: "358371335178617311@lurus-api"
```

### 3. 部署到 K8s

```bash
# 应用 secrets
kubectl apply -f secrets.prod.yaml

# 重启服务以加载新配置
kubectl rollout restart deployment/lurus-api -n lurus-system

# 查看部署状态
kubectl get pods -n lurus-system -w
```

### 4. 验证部署

```bash
# 等待 Pod Running
kubectl wait --for=condition=ready pod -l app=lurus-api -n lurus-system --timeout=120s

# 查看日志
kubectl logs -f deployment/lurus-api -n lurus-system

# 测试 API
curl https://api.lurus.cn/api/status
# 预期: {"status":"ok"}

# 测试版本比较修复（P1-1）
curl "https://api.lurus.cn/api/v1/releases/latest/lurus-cli?current_version=1.9.0"
# 预期: has_update: true (for 1.10.0)

# 测试未登录绑定（P1-2）
curl https://api.lurus.cn/bind/alipay?code=test
# 预期: {"success":false,"message":"未登录或会话已过期"}
```

---

## 📝 详细文档

- **P1 修复详情**: `doc/code-review/2026-02-11-p1-fixes.md`
- **代码审查报告**: `doc/code-review/2026-02-11-code-review.md`
- **部署运维手册**: `doc/runbook/deployment.md`
- **K8s 配置说明**: `deploy/k8s/README.md`

---

## ⚠️ 重要说明

### 数据库密码

**无需更换** - 保持公司标准密码 `LurusOps2026`

虽然密码在 Git 历史中泄露，但这是内部统一标准，不做更改。

### Session Secret

**必须更换** - 旧值 `LurusApiSessionSecret2026Secure!` 已泄露

使用 `openssl rand -base64 32` 生成新值。

### Alipay 密钥

如需配置支付功能，联系支付团队获取：
- Alipay App ID
- RSA2 私钥
- RSA2 公钥

---

## 🔧 故障排查

### Pod 无法启动

```bash
# 查看错误
kubectl describe pod <pod-name> -n lurus-system
kubectl logs <pod-name> -n lurus-system

# 常见问题：
# 1. Secret 格式错误 → 检查 YAML 缩进
# 2. 数据库连接失败 → 检查 SQL_DSN 格式
# 3. 镜像拉取失败 → 检查 GHCR 认证
```

### 数据库连接测试

```bash
# 从 Pod 内测试
kubectl exec -it <pod-name> -n lurus-system -- sh
nc -zv 100.94.177.10 30543

# 或直接测试（需要 psql 客户端）
psql "postgres://lurus:LurusOps2026@100.94.177.10:30543/lurusapi?sslmode=disable" -c "SELECT version()"
```

### Secret 更新未生效

Secret 更新后 **必须重启 Pod**:

```bash
kubectl rollout restart deployment/lurus-api -n lurus-system
```

---

## 🔄 回滚

如果部署出现问题：

```bash
# 回滚到上一个版本
kubectl rollout undo deployment/lurus-api -n lurus-system

# 查看部署历史
kubectl rollout history deployment/lurus-api -n lurus-system
```

---

## 📞 支持

- **代码问题**: 查看 `doc/code-review/2026-02-11-code-review.md`
- **运维问题**: 查看 `doc/runbook/incident-response.md`
- **凭证信息**: 参考 `../../重要信息.md` (内部文档)
