# TTS 项目安全加固指南

## 目录
1. [容器安全](#容器安全)
2. [配置管理](#配置管理)
3. [网络安全](#网络安全)
4. [CI/CD 安全](#cicd-安全)
5. [运行时安全](#运行时安全)
6. [监控与审计](#监控与审计)

---

## 容器安全

### 1. 使用非特权用户

**❌ 安全风险：以 root 运行**
```dockerfile
# 危险！容器默认以 root 用户运行
FROM gcr.io/distroless/static-debian12
COPY /app/main /app/main
# USER 未指定 → 默认 root (UID 0)
```

**✅ 推荐做法：使用 nonroot 用户**
```dockerfile
FROM gcr.io/distroless/static-debian12:nonroot

# 确保文件所有权正确
COPY --chown=nonroot:nonroot /app/main /app/main
COPY --chown=nonroot:nonroot configs /app/configs

# 显式声明用户
USER nonroot:nonroot
ENTRYPOINT ["/app/main"]
```

**验证容器用户：**
```bash
# 检查运行中容器的用户
docker inspect <container_id> | jq '.[0].Config.User'

# 预期输出：nonroot:nonroot 或 65532:65532
```

### 2. 使用最小化基础镜像

**镜像选择原则：**

| 镜像类型 | 适用场景 | 安全评分 |
|---------|---------|---------|
| `scratch` | 完全自包含的应用 | ⭐⭐⭐ |
| `distroless/static` | 静态链接的 Go 应用（**推荐**） | ⭐⭐⭐⭐⭐ |
| `distroless/base` | 需要 glibc 的应用 | ⭐⭐⭐⭐ |
| `alpine` | 需要调试工具的场景 | ⭐⭐⭐ |
| `debian-slim` | 需要完整系统工具 | ⭐⭐ |

**distroless 镜像优势：**
- ✅ 不包含包管理器（无法安装恶意软件）
- ✅ 不包含 shell（无法获得交互式访问）
- ✅ 不包含系统工具（降低攻击面）
- ✅ 包含必要的 CA 证书和时区数据
- ✅ 定期更新安全补丁

### 3. 镜像扫描

**集成 Trivy 扫描到 CI：**
```yaml
security-scan:
  name: Security Scan
  runs-on: ubuntu-latest
  steps:
  - uses: actions/checkout@v4

  - name: Run Trivy vulnerability scanner
    uses: aquasecurity/trivy-action@master
    with:
      scan-type: 'fs'
      scan-ref: '.'
      format: 'sarif'
      output: 'trivy-results.sarif'
      severity: 'CRITICAL,HIGH'

  - name: Upload results to GitHub Security
    uses: github/codeql-action/upload-sarif@v3
    with:
      sarif_file: 'trivy-results.sarif'
```

**本地扫描镜像：**
```bash
# 扫描已构建的镜像
trivy image ch6vip/ch6vip-tts:latest

# 扫描文件系统
trivy fs .

# 生成 SBOM（软件物料清单）
trivy image --format cyclonedx ch6vip/ch6vip-tts:latest > sbom.json
```

### 4. 多阶段构建安全

**确保不泄露构建工具：**
```dockerfile
# ✅ 构建阶段包含编译器和工具
FROM golang:1.23-alpine AS builder
RUN apk add --no-cache git make
COPY . .
RUN go build -o /app/main ./cmd/api

# ✅ 最终镜像仅包含二进制文件
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /app/main /app/main
# 不会将构建工具和源代码带入最终镜像
```

---

## 配置管理

### 1. 敏感信息处理

**❌ 错误做法：硬编码敏感信息**
```yaml
# configs/config.yaml (危险！)
azure:
  speech_key: "your-actual-api-key-here"  # 绝对不要这样做
  endpoint: "https://eastasia.api.cognitive.microsoft.com"
```

**✅ 推荐做法：环境变量注入**
```yaml
# configs/config.yaml (安全)
azure:
  speech_key: "${AZURE_SPEECH_KEY}"  # 运行时从环境变量读取
  endpoint: "${AZURE_ENDPOINT}"
```

**应用代码中的处理：**
```go
// internal/config/config.go
func Load(path string) (*Config, error) {
    cfg := &Config{}
    
    // 加载配置文件
    viper.SetConfigFile(path)
    if err := viper.ReadInConfig(); err != nil {
        return nil, err
    }
    
    // 自动替换环境变量
    viper.AutomaticEnv()
    
    if err := viper.Unmarshal(cfg); err != nil {
        return nil, err
    }
    
    return cfg, nil
}
```

### 2. Docker Secrets

**在 Docker Compose 中使用 Secrets：**
```yaml
version: '3.8'
services:
  tts:
    image: ch6vip/ch6vip-tts:latest
    ports:
      - "127.0.0.1:8080:8080"
    secrets:
      - azure_speech_key
      - api_key
    environment:
      - AZURE_SPEECH_KEY_FILE=/run/secrets/azure_speech_key
      - API_KEY_FILE=/run/secrets/api_key

secrets:
  azure_speech_key:
    file: ./secrets/azure_speech_key.txt
  api_key:
    file: ./secrets/api_key.txt
```

**在应用中读取 Secrets：**
```go
// 读取 Docker Secrets
func readSecret(path string) (string, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return "", err
    }
    return strings.TrimSpace(string(data)), nil
}

// 初始化配置时优先使用 Secrets
if keyFile := os.Getenv("AZURE_SPEECH_KEY_FILE"); keyFile != "" {
    key, err := readSecret(keyFile)
    if err == nil {
        cfg.Azure.Key = key
    }
}
```

### 3. Kubernetes Secrets

```yaml
# k8s-secrets.yaml
apiVersion: v1
kind: Secret
metadata:
  name: tts-secrets
type: Opaque
stringData:
  azure-speech-key: "your-api-key"
  api-key: "your-api-key"
---
apiVersion: v1
kind: Pod
metadata:
  name: tts
spec:
  containers:
  - name: tts
    image: ch6vip/ch6vip-tts:latest
    env:
    - name: AZURE_SPEECH_KEY
      valueFrom:
        secretKeyRef:
          name: tts-secrets
          key: azure-speech-key
    - name: API_KEY
      valueFrom:
        secretKeyRef:
          name: tts-secrets
          key: api-key
```

---

## 网络安全

### 1. 限制暴露端口

**❌ 不安全的端口绑定：**
```yaml
# docker-compose.yml
ports:
  - "8080:8080"  # 危险！暴露在所有网络接口
```

**✅ 绑定到本地回环地址：**
```yaml
# docker-compose.yml
ports:
  - "127.0.0.1:8080:8080"  # 安全：仅本地访问
```

### 2. 使用反向代理

**Nginx 反向代理配置：**
```nginx
# nginx.conf
upstream tts_backend {
    server 127.0.0.1:8080;
}

server {
    listen 443 ssl http2;
    server_name tts.example.com;

    ssl_certificate /etc/nginx/ssl/cert.pem;
    ssl_certificate_key /etc/nginx/ssl/key.pem;

    # 安全头部
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Strict-Transport-Security "max-age=31536000" always;

    # 速率限制
    limit_req_zone $binary_remote_addr zone=api_limit:10m rate=10r/s;
    limit_req zone=api_limit burst=20 nodelay;

    location / {
        proxy_pass http://tts_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### 3. 网络策略（Kubernetes）

```yaml
# network-policy.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: tts-network-policy
spec:
  podSelector:
    matchLabels:
      app: tts
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app: nginx-ingress
    ports:
    - protocol: TCP
      port: 8080
  egress:
  - to:
    - namespaceSelector: {}
    ports:
    - protocol: TCP
      port: 443  # 允许访问 Azure API
```

---

## CI/CD 安全

### 1. Secrets 管理

**GitHub Actions Secrets 配置：**
```yaml
# .github/workflows/ci.yml
- name: Log in to Docker Hub
  uses: docker/login-action@v3
  with:
    username: ${{ secrets.DOCKER_USERNAME }}
    password: ${{ secrets.DOCKER_PASSWORD }}
```

**创建 Secrets：**
```bash
# 通过 GitHub CLI
gh secret set DOCKER_USERNAME
gh secret set DOCKER_PASSWORD

# 或通过 Web UI：
# Settings → Secrets and variables → Actions → New repository secret
```

### 2. 限制工作流权限

```yaml
# .github/workflows/ci.yml
permissions:
  contents: read          # 只读代码
  packages: write         # 写入容器镜像
  security-events: write  # 上传安全扫描结果
```

### 3. 签名镜像（Cosign）

```yaml
# 在 CI 中签名镜像
- name: Install Cosign
  uses: sigstore/cosign-installer@v3

- name: Sign the image
  run: |
    cosign sign --yes \
      -a "repo=${{ github.repository }}" \
      -a "workflow=${{ github.workflow }}" \
      ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}@${{ steps.build.outputs.digest }}
  env:
    COSIGN_EXPERIMENTAL: 1
```

**验证签名：**
```bash
cosign verify ch6vip/ch6vip-tts:latest
```

---

## 运行时安全

### 1. 容器安全上下文

**Docker Compose 安全配置：**
```yaml
version: '3.8'
services:
  tts:
    image: ch6vip/ch6vip-tts:latest
    ports:
      - "127.0.0.1:8080:8080"
    security_opt:
      - no-new-privileges:true  # 禁止提升权限
    cap_drop:
      - ALL                     # 移除所有 Linux 能力
    cap_add:
      - NET_BIND_SERVICE        # 仅添加必要的能力
    read_only: true             # 只读文件系统
    tmpfs:
      - /tmp                    # 临时文件系统
    restart: always
```

**Kubernetes SecurityContext：**
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: tts
spec:
  securityContext:
    runAsNonRoot: true
    runAsUser: 65532
    fsGroup: 65532
    seccompProfile:
      type: RuntimeDefault
  containers:
  - name: tts
    image: ch6vip/ch6vip-tts:latest
    securityContext:
      allowPrivilegeEscalation: false
      readOnlyRootFilesystem: true
      capabilities:
        drop:
        - ALL
```

### 2. 资源限制

```yaml
# docker-compose.yml
services:
  tts:
    image: ch6vip/ch6vip-tts:latest
    deploy:
      resources:
        limits:
          cpus: '1.0'
          memory: 512M
        reservations:
          cpus: '0.5'
          memory: 256M
```

---

## 监控与审计

### 1. 日志记录

**应用日志配置：**
```go
// cmd/api/main.go
func initLog(logConfig *config.LogConfig) {
    // 使用结构化日志（JSON 格式）
    logrus.SetFormatter(&logrus.JSONFormatter{
        TimestampFormat: time.RFC3339,
        FieldMap: logrus.FieldMap{
            logrus.FieldKeyTime:  "timestamp",
            logrus.FieldKeyLevel: "level",
            logrus.FieldKeyMsg:   "message",
        },
    })
    
    // 记录关键安全事件
    logrus.AddHook(&SecurityHook{})
}
```

### 2. 审计日志

**记录关键操作：**
```go
// 记录 API 访问
logrus.WithFields(logrus.Fields{
    "event": "api_access",
    "user": user.ID,
    "endpoint": r.URL.Path,
    "method": r.Method,
    "ip": r.RemoteAddr,
}).Info("API request")

// 记录配置变更
logrus.WithFields(logrus.Fields{
    "event": "config_change",
    "key": "azure.endpoint",
    "old_value": oldValue,
    "new_value": newValue,
}).Warn("Configuration updated")
```

### 3. 异常检测

**监控异常行为：**
```go
// 检测大量失败请求
if failureRate > 0.1 {
    logrus.WithFields(logrus.Fields{
        "event": "security_alert",
        "alert_type": "high_failure_rate",
        "failure_rate": failureRate,
    }).Error("Potential attack detected")
}
```

---

## 安全检查清单

### 容器安全
- [ ] 使用非 root 用户运行容器
- [ ] 使用最小化基础镜像（distroless）
- [ ] 定期扫描镜像漏洞
- [ ] 签名和验证容器镜像
- [ ] 使用只读文件系统
- [ ] 限制容器能力（capabilities）

### 配置安全
- [ ] 不在代码或配置文件中硬编码敏感信息
- [ ] 使用环境变量或 Secrets 管理敏感数据
- [ ] 定期轮换密钥和令牌
- [ ] 加密配置文件（如有必要）

### 网络安全
- [ ] 仅暴露必要的端口
- [ ] 使用反向代理和 TLS
- [ ] 实施速率限制和 DDoS 防护
- [ ] 配置网络策略（Kubernetes）

### CI/CD 安全
- [ ] 使用 GitHub Secrets 管理凭证
- [ ] 限制工作流权限
- [ ] 集成安全扫描（Trivy、CodeQL）
- [ ] 使用受信任的第三方 Actions

### 运行时安全
- [ ] 配置安全上下文
- [ ] 设置资源限制
- [ ] 启用审计日志
- [ ] 监控异常行为

---

## 参考资源

- [OWASP Docker Security Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Docker_Security_Cheat_Sheet.html)
- [CIS Docker Benchmark](https://www.cisecurity.org/benchmark/docker)
- [Kubernetes Security Best Practices](https://kubernetes.io/docs/concepts/security/)
- [Distroless Container Images](https://github.com/GoogleContainerTools/distroless)
- [Trivy Vulnerability Scanner](https://aquasecurity.github.io/trivy/)