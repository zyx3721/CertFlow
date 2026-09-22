<div align="center">

<h1>CertFlow</h1>

<p><b>企业内部 PKI 证书生命周期管理平台</b> — CA 分级 · 证书签发 · 审批 · 撤销 · CRL · OCSP · 审计</p>

<p><b>简体中文</b> · <a href="README.en.md">English</a></p>

内部业务的 HTTPS 证书、客户端证书、mTLS 双向认证证书，通常散在运维的 U 盘、Excel 和口头审批里。  
CertFlow 把它们收进一套**自托管**的信任体系：Go 后端 + React 控制台 + PostgreSQL。  
CA 分级、证书申请审批、撤销与 CRL/OCSP 发布一条龙，私钥全程加密落库；  
仓库里没有 Token、没有密码、没有真实主机名。

<p>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue?labelColor=1f2937" alt="MIT License"></a>
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white&labelColor=1f2937" alt="Go 1.25+"></a>
  <a href="https://react.dev/"><img src="https://img.shields.io/badge/React-19-61DAFB?logo=react&logoColor=white&labelColor=1f2937" alt="React 19"></a>
  <a href="https://www.postgresql.org/"><img src="https://img.shields.io/badge/PostgreSQL-16+-4169E1?logo=postgresql&logoColor=white&labelColor=1f2937" alt="PostgreSQL"></a>
  <img src="https://img.shields.io/badge/%E6%9D%83%E9%99%90-27%20%E9%A1%B9-059669?labelColor=1f2937" alt="27 项权限">
</p>

<p>
  <b><a href="#项目预览">项目预览</a></b> ·
  <a href="#它做什么">它做什么</a> ·
  <a href="#怎么工作">怎么工作</a> ·
  <a href="#技术栈">技术栈</a> ·
  <a href="#快速开始">快速开始</a> ·
  <a href="#部署">部署</a> ·
  <a href="#权限模型">权限</a> ·
  <a href="#数据与安全">安全</a> ·
  <a href="#常见问题">常见问题</a> ·
  <a href="#项目结构">项目结构</a> ·
  <a href="#文档">文档</a>
</p>

</div>

---

## 项目预览

### 登录

内部系统，没有公开注册。支持本地密码、AD/LDAP 与企业微信扫码登录（直连或统一认证中心模式），找回密码走图形验证码 + 邮箱验证码。

![登录页](.github/images/certflow-login.jpg)

### 首页

仪表盘只读汇总：CA 与证书总量、签发/撤销趋势与近期审计活动。侧栏按登录用户的权限逐项显示或隐藏，无权操作不会出现在界面上。

![首页](.github/images/certflow-home.jpg)

## 它做什么

- **CA 管理** — 创建或导入 Root、Intermediate、Issuing 三级 CA，支持 RSA、ECDSA、ED25519 算法；CA 树可视化维护，查看与下载 PEM 证书；只有未关联证书的 CA 分支才允许递归删除，保证签发链始终可追溯。
- **证书生命周期** — Subject、SAN、用途（服务器/客户端/mTLS）、有效期与可选 CSR 的证书申请；提交即生成审批单，批准后自动生成序列号并签发；材料下载按状态返回 CRT、证书链 ZIP 或被驳回申请的 CSR；支持状态校验、按标准原因撤销与删除。
- **CRL 与 OCSP** — 标准信任链的 X.509 CRL 即时签名下载（单 CA `.crl` 或多 CA 打包 ZIP）；RFC 6960 二进制 OCSP 响应由签发 CA 对应的独立 Responder 证书签名，另有 JSON 状态查询接口；撤销历史持久保留，支持 CRL 与 OCSP 的持续校验。
- **审批与审计** — 证书申请自动创建审批记录，批准/驳回全程留痕；登录注销、CA、证书、审批、用户、系统配置等关键操作全量审计，支持搜索、筛选与导出。
- **到期提醒与自动续期** — 按五段 Cron 定时扫描，阈值内证书向邮件、Webhook、飞书、企业微信、钉钉媒介推送提醒；启用自动续期后在续期窗口内直接签发使用新密钥材料的续期证书。
- **用户与权限** — 本地密码、AD/LDAP、企业微信扫码登录（直连或统一认证中心 SSO 模式）、企微账号绑定、找回密码邮件流程；内置 `admin` / `operator` / `viewer` 三角色，自定义角色可从 27 项权限中勾选，并支持用户群组批量授权。前端隐藏无权入口，后端逐接口校验。
- **系统配置** — 品牌标识、CRL/OCSP 服务参数、续期与到期提醒、找回密码安全时效、企业微信扫码有效期、用户/用户群组/角色、AD/LDAP 与企业微信认证、通知媒介配置，并提供 LDAP 连通性与邮件发送测试。
- **API 文档** — 集成 swag 与 Swagger UI，接口、参数、鉴权与响应定义开箱可查。

**它不是**证书透明度日志，也不是扫描器：CertFlow 管的是「内部 CA + 证书台账 + 审批 + 撤销证明」，不扫描网络内的证书，不做 ACME 自动签发，不代理外部公有云 KMS。

同一本内部证书账，放在两种做法里大致是这样：

| 场景 | 手工管理 | 用 CertFlow |
| --- | --- | --- |
| 签一张业务证书 | 运维私底下用 openssl 敲命令 | 表单填写 Subject/SAN，提交后走审批 |
| 谁批的 | 聊天记录里翻 | 审批单与审计日志全量留痕 |
| 证书快过期 | 靠自己记得 | 阈值内自动提醒，可自动续期 |
| 私钥被要走 | 压缩包满天飞 | 私钥 AES-256-GCM 加密落库，专用接口授权下载 |
| 吊销了怎么证明 | 发邮件通知各业务方 | CRL/OCSP 标准分发，业务方自己查 |
| CA 私钥 | 存在某个人的电脑里 | 加密落库，接口永不回显 |
| 权限 | 一把梭 | 27 项权限逐接口校验 |

## 怎么工作

```text
        浏览器
           │  http
           ▼
  ┌──────────────────────────────────────┐
  │  Nginx（容器内或宿主机）                 │
  │  /assets/ · /       → 前端 SSR         │
  │  /api/ · /swagger/  → Go 后端          │
  │  /crl/ · /ocsp      → Go 后端（公开）   │
  └──────────────────────────────────────┘
        │                        │
        ▼                        ▼
  React 19 控制台            Go 1.25 后端
  Nitro SSR :5173       net/http :8080
                                 │
                          PostgreSQL 16+
                     用户 · CA · 证书 · 审计
```

- **谁负责什么** — 控制台读取数据库当前状态展示 CA、证书与统计；CA 创建、证书申请、审批、撤销、系统配置等变更由后端执行权限校验并写入审计记录。
- **数据放哪** — PostgreSQL 是业务数据的事实来源，启动时按文件名顺序执行嵌入式迁移；证书私钥以应用主密钥加密后落库。
- **敏感信息** — 会话令牌仅以 SHA-256 哈希保存；私钥、LDAP Bind Password、SMTP Password 与企业微信 Secret 加密落库，接口返回时只暴露是否已配置。
- **对外暴露** — 只有健康检查、品牌信息、登录、企微授权回调、找回密码流程与 CRL/OCSP 公开分发可匿名访问，其余接口一律要求 `Authorization: Bearer <token>`。

## 技术栈

| 层 | 选型 |
| --- | --- |
| 后端语言 | Go 1.25+ |
| HTTP 路由 | Go 标准库 `net/http` 与 `ServeMux` |
| 数据库 | PostgreSQL 16+（[pgx/v5](https://github.com/jackc/pgx) 连接池） |
| 认证与加密 | 不透明会话令牌 + bcrypt、AD/LDAP（go-ldap/ldap v3）、企业微信 OAuth 扫码登录（直连 / 统一认证中心 SSO）、AES-256-GCM 敏感配置加密 |
| PKI 能力 | Go 标准库 `crypto/x509`、`golang.org/x/crypto/ocsp`、CRL 即时签名 |
| API 文档 | swag + http-swagger（Swagger UI） |
| 前端框架 | React 19 + TanStack Start / Router / Query |
| 语言与构建 | TypeScript 5 + Vite + Nitro |
| 样式与组件 | Tailwind CSS、Radix UI、lucide-react、sonner |
| 运行时打包 | Docker（Nginx + Supervisor 多进程） |

## 快速开始

本地开发需要 **Go 1.25+**、**Node.js 20+** 与 **PostgreSQL 16+**。

```bash
git clone https://github.com/zyx3721/CertFlow.git
cd CertFlow
```

**准备数据库**

```bash
psql -Upostgres -c "CREATE DATABASE certflow;"
```

没有现成 PostgreSQL 时，可用 Docker 快速起一个：

```bash
docker run -d --name pg-prod \
  -p 5432:5432 \
  -v /data/PgSqlData:/var/lib/postgresql/data \
  -e POSTGRES_PASSWORD="123456ok!" \
  -e LANG=C.UTF-8 -e TZ=Asia/Shanghai \
  postgres:17-alpine
```

**后端**

```bash
cd backend
go mod download
cp .env.example .env      # 配置数据库连接；生成密钥：openssl rand -base64 32
go run cmd/server/main.go
```

后端默认监听 `http://localhost:8080`，首次启动自动执行数据库迁移并创建默认管理员 `admin / 123456`。

**前端**（另开一个终端）

```bash
cd frontend
npm install
npm run dev
```

前端默认运行在 `http://localhost:5173`（后端端口不是 8080 时，按 `frontend/.env` 的 `VITE_API_BASE_URL` 指定）。

打开 `http://localhost:5173`，使用 `admin / 123456` 登录，**登录后立刻改密码**。Swagger 在 `http://localhost:8080/swagger/index.html`。

## 部署

只保留两种方式：**Docker Compose 部署**（推荐）与 **Release 二进制部署**。完整过程化步骤（含宿主机 Nginx HTTP/HTTPS 示例）见 [docs/manual.md](docs/manual.md)。

### 方式一：Docker Compose 部署

镜像内置 Go 后端、Nginx 与前端 Nitro SSR，由 Supervisor 管理多进程，对外只暴露 80 端口：

```bash
git clone https://github.com/zyx3721/CertFlow.git
cd CertFlow/deploy
cp .env.example .env && vim .env    # 配置数据库与 JWT_SECRET、PKI_KEY_ENCRYPTION_KEY
docker compose up -d
```

`.env` 关键项（完整参数见 [docs/manual.md](docs/manual.md) 5.5 节）：

| 变量 | 说明 |
| --- | --- |
| `JWT_SECRET` | 会话签名密钥；生产环境必须显式设置足够随机的长字符串 |
| `PKI_KEY_ENCRYPTION_KEY` | 私钥与敏感配置加密主密钥，Base64 编码 32 字节；`openssl rand -base64 32` 生成，**密钥丢失则已加密私钥不可恢复** |
| `DB_HOST` / `DB_PASSWORD` 等 | PostgreSQL 连接；默认随 compose 启动 `postgres` 容器，使用外部库时注释掉该服务块 |

服务管理：

```bash
docker compose ps                    # 查看运行状态
docker compose logs -f certflow      # 查看实时日志
docker compose restart certflow      # 重启
docker compose down                  # 停止
```

**访问**

- 控制台：`http://your-host/`，默认账号 `admin / 123456`
- 接口文档：`http://your-host/swagger/index.html`
- 健康检查：`http://your-host/health`

需要在宿主机上统一做域名、HTTPS 或多站点入口时，把端口映射改为非 80 端口（`8080:80`），再由宿主机 Nginx 反代到容器。注意 `/crl/`、`/ocsp` 等公开 PKI 分发路径必须一并反代到后端。

### 方式二：Release 二进制部署

前往 [GitHub Releases](https://github.com/zyx3721/CertFlow/releases) 页面，按自己的操作系统与 CPU 架构下载对应压缩包，再按下面步骤校验、解压、配置、启动。

**下载哪个包**

| 你的机器 | 下载文件 |
| --- | --- |
| Linux x86_64 | `certflow_<版本>_linux_amd64.tar.gz` |
| Linux ARM64（鲲鹏、飞腾等） | `certflow_<版本>_linux_arm64.tar.gz` |
| macOS Intel 芯片 | `certflow_<版本>_darwin_amd64.tar.gz` |
| macOS Apple 芯片 | `certflow_<版本>_darwin_arm64.tar.gz` |
| Windows x86_64 | `certflow_<版本>_windows_amd64.zip` |
| Windows ARM64 | `certflow_<版本>_windows_arm64.zip` |
| 前端界面（以上任意平台都需要） | `certflow-frontend_<版本>.tar.gz` |
| 校验和 | `SHA256SUMS` |

后端包内是 `certflow` 可执行文件（Windows 为 `certflow.exe`）、`.env.example` 与 `README.txt`；前端包内是 Nitro SSR 的 `.output` 产物。后端二进制无运行时依赖；前端 SSR 需要目标机器上安装 Node.js，且无论哪种方式都需要自备 PostgreSQL 16+。

**1. 校验下载**

```bash
VERSION=1.0.3
mkdir -p /data/certflow && cd /data/certflow
sha256sum -c SHA256SUMS
```

**2. 解压**

```bash
mkdir -p backend frontend/.output
tar -xzf certflow_${VERSION}_linux_amd64.tar.gz -C backend --strip-components=1
tar -xzf certflow-frontend_${VERSION}.tar.gz -C frontend/.output
```

得到的目录结构：

```text
/data/certflow/
├── backend/
│   ├── certflow           # 后端二进制
│   └── .env.example
└── frontend/
    └── .output/
        ├── public/        # 浏览器静态资源
        └── server/
            └── index.mjs  # Nitro SSR 入口
```

**3. 配置并启动后端**

```bash
cd /data/certflow/backend
cp .env.example .env
vim .env               # 至少设置 JWT_SECRET 与 PKI_KEY_ENCRYPTION_KEY，并指向可用的 PostgreSQL
./certflow
```

后端同时支持命令行参数，显式传入的参数优先于环境变量与 `.env` 文件；`./certflow -v` 可查看版本信息（版本、commit、构建时间），`./certflow -h` 查看全部参数：

| 参数 | 等价环境变量 | 说明 |
| --- | --- | --- |
| `-host` | `SERVER_HOST` | 后端监听地址 |
| `-port` | `SERVER_PORT` | 后端监听端口 |
| `-mode` | `SERVER_MODE` | 运行模式（release/dev） |
| `-db-host` | `DB_HOST` | PostgreSQL 主机 |
| `-db-port` | `DB_PORT` | PostgreSQL 端口 |
| `-db-name` | `DB_NAME` | 数据库名 |
| `-db-user` | `DB_USER` | 数据库用户 |
| `-db-password` | `DB_PASSWORD` | 数据库密码 |
| `-db-sslmode` | `DB_SSLMODE` | 数据库 SSL 模式 |
| `-jwt-secret` | `JWT_SECRET` | 会话签名密钥 |
| `-session-ttl` | `JWT_EXPIRE_HOURS` | 会话有效期（小时） |
| `-pki-key` | `PKI_KEY_ENCRYPTION_KEY` | 私钥加密主密钥（Base64 编码 32 字节） |
| `-cors-origin` | `CORS_ORIGIN` | 允许的跨域来源 |
| `-env` | 无 | 指定 `.env` 配置文件路径 |
| `-v`、`-version` | 无 | 显示版本信息并退出 |

需要常驻时交给 systemd：

```ini
# /etc/systemd/system/certflow-backend.service
[Unit]
Description=CertFlow Backend
After=network.target

[Service]
Type=simple
WorkingDirectory=/data/certflow/backend
ExecStart=/data/certflow/backend/certflow
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

```bash
systemctl daemon-reload && systemctl enable --now certflow-backend
```

**4. 启动前端 SSR**

```bash
cd /data/certflow/frontend
SSR_API_ORIGIN=http://127.0.0.1:8080 HOST=127.0.0.1 PORT=5173 node .output/server/index.mjs
```

SSR 进程启动时会访问后端拉取品牌配置，把站点名与图标直出进首帧 HTML。后端地址按以下顺序确定：运行时环境变量 `SSR_API_ORIGIN` → 入口文件 `index.mjs` 所在目录向上任意一层的 `.env` 文件（首个存在的生效，可与后端共用部署根目录的同一份 `.env`）→ 默认 `http://127.0.0.1:8080`。请确保该地址对 SSR 进程可达，否则首屏会先显示默认品牌、加载后再切换为配置值，SSR 进程日志会输出 `[certflow-ssr] fetch brand failed` 警告。

**5. 用 Nginx 收口**

```nginx
server {
    listen 80;
    server_name your-domain.com;
    client_max_body_size 50m;

    # 前端静态资源：直接读取 .output/public
    location ^~ /assets/ {
        root /data/certflow/frontend/.output/public;
        try_files $uri =404;
        expires 1y;
        add_header Cache-Control "public, immutable";
    }

    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location /swagger/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
    }

    # 公开 PKI 分发：CRL 与 OCSP 必须保留原始路径与方法反代到后端
    location ^~ /crl/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        add_header X-Content-Type-Options nosniff always;
    }

    location = /ocsp {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        add_header X-Content-Type-Options nosniff always;
    }

    location ^~ /ocsp/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        add_header X-Content-Type-Options nosniff always;
    }

    location / {
        proxy_pass http://127.0.0.1:5173;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location = /health {
        proxy_pass http://127.0.0.1:8080/api/health;
    }
}
```

前端必须经 `node .output/server/index.mjs` 提供 SSR；**只把 `.output/public` 配成静态根目录会导致服务端渲染页面无法返回**。含 HTTPS 与 80→443 跳转的完整示例见 [docs/manual.md](docs/manual.md) 4.4 节。

**6. 访问**

同 Docker 方式：控制台 `http://your-domain.com`（`admin / 123456`）、接口文档 `/swagger/index.html`、健康检查 `/health`。

## 权限模型

接口按角色权限逐个校验，无权限返回 403；`admin` 用户拥有全部权限。权限 Key 形如 `模块.资源.操作`，共 27 项。

| 身份 | 默认权限 |
| --- | --- |
| 默认管理员 `admin` | 全部 27 项；默认账号不可删除 |
| 内置角色 `admin` | 全部 27 项 |
| 内置角色 `operator` | 仪表盘、CA、证书与审批的完整操作，审计查看与导出；不含任何系统配置权限 |
| 内置角色 `viewer` | 只读：仪表盘、CA、证书、CRL 与审计查看 |
| 自定义角色 / 用户群组 | 从 27 项中勾选；管理权限自动补全对应读取权限，用户群组可批量授权 |

按模块划分：

- **仪表盘** — `dashboard.read`
- **CA 管理** — `ca.{read|download|add|delete}`
- **证书管理** — `certificates.{read|request|download|verify|manage|revoke|delete}`；申请证书自动附带查看证书与查看 CA
- **审批管理** — `workflows.{read|approve|delete}`
- **证书撤销** — `crl.{read|manage}`
- **审计日志** — `audit.{read|manage}`（导出，隐含查看）
- **系统配置** — `settings.{base|users|auth|notifications}.read` / `.manage`

## 数据与安全

```text
控制台账号登录后台、改配置、管证书
        +
Bearer Token 逐接口校验 27 项权限，无权限 403
        +
私钥与敏感配置 AES-256-GCM 加密落库，接口回显脱敏
        +
仓库禁止提交 Token / 密码 / 真实主机名
```

- **先改默认密码** — 首次部署后立即修改 `admin` 的默认口令。
- **显式设置两个密钥** — `JWT_SECRET` 未设置时后端仅生成进程内临时密钥，重启即全体会话失效；`PKI_KEY_ENCRYPTION_KEY` 错误或缺失时后端拒绝启动，且它加密的私钥无法恢复，务必妥善备份。
- **启用 HTTPS** — 生产环境通过 Nginx 配置证书，完整示例见 [docs/manual.md](docs/manual.md) 4.4.2 节。
- **收紧跨域** — 生产环境按需配置 `CORS_ORIGIN`，不要保留 `*`。
- **私钥边界** — 私钥不出现在 CA 或证书列表 API；下载必须使用专用、授权的下载接口，且审计留痕。
- **认证安全** — LDAP Bind Password、SMTP Password 与企业微信 Secret（直连应用 Secret、统一认证中心应用 Secret）同主密钥加密保存，API 只回显是否已配置；企微授权 state 以会话密钥 HMAC-SHA256 签名（默认 5 分钟、1-60 分钟可调），统一认证中心 ticket 取出即删、HMAC 签名 + 时间戳窗口校验。
- **审计可追溯** — 登录（含企微扫码、失败归因）、绑定/解绑、CA、证书、审批、用户与系统配置变更全量记录操作时间、用户、模块、目标、来源 IP、结果与非敏感详情。

## API 文档

后端集成 Swagger/OpenAPI，启动后即可查看在线接口文档：

- **Swagger UI**：`http://localhost:8080/swagger/index.html`
- **OpenAPI JSON**：`http://localhost:8080/swagger/doc.json`
- **健康检查**：`GET /api/health`

无需认证的接口只有：`GET /api/v1/auth/providers`、`GET /api/v1/auth/wecom/authorize`、`POST /api/v1/auth/wecom/callback`、`POST /api/v1/auth/wecom/sso/callback`、`GET /api/v1/auth/password-reset/captcha`、`POST /api/v1/auth/password-reset/verify`、`POST /api/v1/auth/password-reset/send`、`POST /api/v1/auth/password-reset/confirm`、`GET /api/v1/public/settings`、`GET /api/health`，以及公开 PKI 分发 `GET /crl/{caId}.crl`、`POST /ocsp`、`GET /ocsp`、`GET /ocsp/{serial}`、`GET /ocsp/health`；其余接口均需在请求头携带 `Authorization: Bearer <token>`。

登录请求示例：

```json
{
  "username": "admin",
  "password": "123456",
  "provider": "local"
}
```

按模块分组的完整接口清单（健康检查、身份认证、找回密码、CA、证书、CRL、审批、审计、系统配置、公开 PKI 服务）见 [docs/manual.md](docs/manual.md) 的《七、API 文档》。

修改接口后，在 `backend/` 目录执行以下命令同步 Swagger 产物：

```bash
swag init -g cmd/server/main.go -o docs
```

## 数据库

PostgreSQL 数据库，共 20 张表，由嵌入式迁移按文件名顺序创建（`backend/pkg/database/migrations/`），已应用的迁移记录在 `schema_migrations` 不会重放。

| 分组 | 表 |
| --- | --- |
| 用户与会话 | `users`、`sessions`、`roles`、`user_roles`、`user_groups`、`user_group_members`、`user_group_roles`、`user_wecom_bindings` |
| PKI 业务 | `certificate_authorities`、`certificates`、`certificate_revocation_entries`、`ocsp_responders`、`workflows` |
| 系统与配置 | `system_settings`、`auth_provider_settings`、`notification_channel_settings`、`certificate_expiry_notifications` |
| 审计与找回密码 | `audit_entries`、`password_reset_requests` |

每张表的字段与用途见 [docs/manual.md](docs/manual.md) 的 `001_init.sql` 相关章节。

## 常见问题

**忘记管理员密码怎么办？**

直接改数据库即可（推荐，不丢数据）。先用任意 bcrypt 工具为新口令生成哈希（例如 `123456` 对应 `$2a$10$y6sEolCX.y.We871sMtwkO2MkT4dUUJhafNtMZeLwbT07DV62JkuS`），再更新：

```bash
psql -Upostgres -d certflow -c 'UPDATE users SET password_hash = ''$2a$10$y6sEolCX.y.We871sMtwkO2MkT4dUUJhafNtMZeLwbT07DV62JkuS'' WHERE username = ''admin'';'
```

**会话有效期多久？**

默认 12 小时（`JWT_EXPIRE_HOURS`），本地密码、LDAP 与企业微信登录统一生效，修改后重启后端生效。会话令牌哈希落库，重设 `JWT_SECRET` 不会使已登录会话失效；需要强制下线全部用户时清空 `sessions` 表。

**怎么启用 LDAP 登录？**

第一步在「系统配置 → 认证配置 → AD/LDAP」填好连接参数并启用（可用「测试」按钮验证连通性与匹配用户数）；第二步在「用户配置」中创建与 LDAP `sAMAccountName` 同名的用户。本地用户表中不存在该用户名时，即使 LDAP 密码正确也无法登录。

**怎么启用企业微信扫码登录？**

在企业微信管理后台「应用管理」创建自建应用，记录 AgentID 与 Secret；在应用的「网页授权及 JS-SDK」中将本系统访问域名配置为可信回调域名，并在「企业可信 IP」中加入本服务出口 IP。再到「系统配置 → 认证配置 → 企业微信」选择「直连企业微信」方式，填写企业 ID（corpid）、AgentID、Secret 并启用（回调地址前缀可留空，按当前访问地址自动推断）。用户先用账号密码登录，在右上角菜单「绑定企微」扫码完成账号关联，之后即可在登录页选择企业微信扫码登录。

若企业内部已部署统一认证中心（wecom-auth-center），可将「认证方式」切换为「统一认证中心」，填写认证中心地址、应用标识与应用密钥；认证中心侧为本系统配置 `domain`（本系统外部访问地址）与 `callback_path`（`/login`）即可，多个内部系统可共用同一套企微应用配置，绑定与登录交互保持一致。

**数据库能直接拷走迁移吗？**

可以。停服后使用 `pg_dump` / `pg_restore`（或 `CREATE DATABASE ... TEMPLATE`）整体迁移；应用侧只需保持 `JWT_SECRET` 与 `PKI_KEY_ENCRYPTION_KEY` 一致，否则私钥无法解密。

**为什么前端构建后不能只配静态目录？**

因为控制台是 TanStack Start + Nitro 的 SSR 应用，页面由 `node .output/server/index.mjs` 返回，静态目录只提供 `/assets/` 等资源。

其余部署细节见 [docs/manual.md](docs/manual.md) 与 [新手证书签发与 Nginx 配置指南](docs/CertFlow新手证书签发与Nginx配置指南.md)。

## 项目结构

```text
CertFlow/
├── backend/                 Go API、PKI 服务和 PostgreSQL 迁移
│   ├── api/router/          HTTP 边界、认证和权限校验、Swagger 注解与文档模型
│   ├── config/              环境变量加载与校验
│   ├── docs/                Swagger/OpenAPI 生成文件
│   ├── internal/            领域模型、仓储、安全和业务服务
│   ├── pkg/database/        PostgreSQL 连接、初始化与数据库迁移
│   └── cmd/server/          服务入口
├── deploy/                  Docker Compose、Nginx 与进程管理部署文件
├── docs/                    完整版说明文档与新手指南
├── frontend/                TanStack Start React 控制台
│   └── src/
│       ├── components/      应用布局、通知中心、弹窗与基础 UI
│       ├── features/        认证、PKI 业务、系统配置页面与角色编辑组件
│       ├── lib/             认证、PKI、系统配置 API 客户端与品牌、环境变量工具
│       ├── routes/          TanStack Router 路由薄层
│       ├── router.tsx       路由实例
│       ├── start.ts         React Start 入口
│       └── server.ts        服务端入口与错误处理
├── .github/                 GitHub Actions 工作流与预览图
├── .dockerignore            Docker 构建忽略规则
├── .gitignore               Git 忽略规则
├── LICENSE
├── README.md                中文说明（本文件）
├── README.en.md             English
└── docs/manual.md           完整版说明（含全量接口清单、Nginx 与 HTTPS 示例）
```

## 文档

| 先看这个 | 再往下 |
| --- | --- |
| [快速开始](#快速开始) | 本地起后端与前端，默认账号与端口 |
| [部署](#部署) | Docker Compose 与 Release 二进制两条路径、环境变量、反向代理 |
| [权限模型](#权限模型) | 27 项权限怎么分组、内置角色各有什么 |
| [完整版说明](docs/manual.md) | 全量接口清单、Nginx 与 HTTPS 完整示例、环境变量明细 |
| [新手指南](docs/CertFlow新手证书签发与Nginx配置指南.md) | 从创建内部 CA 到签发业务证书并部署 Nginx HTTPS |
| [English README](README.en.md) | 同样的内容，英文版 |

## 版本历史

| 版本 | 发布日期 | 更新日志 |
| --- | --- | --- |
| v1.1.3 | 2026-09-23 | [verchanglog/v1.1.3.md](verchanglog/v1.1.3.md) |
| v1.1.2 | 2026-09-22 | [verchanglog/v1.1.2.md](verchanglog/v1.1.2.md) |
| v1.1.1 | 2026-09-22 | [verchanglog/v1.1.1.md](verchanglog/v1.1.1.md) |
| v1.1.0 | 2026-09-20 | [verchanglog/v1.1.0.md](verchanglog/v1.1.0.md) |
| v1.0.3 | 2026-07-28 | [verchanglog/v1.0.3.md](verchanglog/v1.0.3.md) |
| v1.0.2 | 2026-07-26 | [verchanglog/v1.0.2.md](verchanglog/v1.0.2.md) |
| v1.0.1 | 2026-07-26 | [verchanglog/v1.0.1.md](verchanglog/v1.0.1.md) |
| v1.0.0 | 2026-07-26 | [verchanglog/v1.0.0.md](verchanglog/v1.0.0.md) |

各版本的构建产物与发布说明见 [GitHub Releases](https://github.com/zyx3721/CertFlow/releases)。

## 致谢

感谢以下开源项目与技术社区：

- [jackc/pgx](https://github.com/jackc/pgx) — PostgreSQL 高性能驱动与连接池
- [swaggo/swag](https://github.com/swaggo/swag) — Swagger 文档生成工具
- [go-ldap/ldap](https://github.com/go-ldap/ldap) — LDAP v3 客户端
- [TanStack](https://tanstack.com/) — Router / Query / Start 前端框架套件
- [Tailwind CSS](https://tailwindcss.com/) — 原子化 CSS 框架

## 许可证

本项目采用 [MIT License](LICENSE) 开源协议，可自由使用、复制、修改、合并、发布、分发、再许可与销售，只需在所有副本或重要部分中保留版权声明与许可声明。

## 联系方式

- **Email**：416685476@qq.com
- **GitHub Issues**：[zyx3721/CertFlow/issues](https://github.com/zyx3721/CertFlow/issues)
- **项目主页**：[github.com/zyx3721/CertFlow](https://github.com/zyx3721/CertFlow)

---

**⭐ 如果这个项目对您有帮助，欢迎 Star 支持！**
